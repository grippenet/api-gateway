package v1

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/coneno/logger"
	"github.com/gin-gonic/gin"
	"github.com/influenzanet/go-utils/pkg/api_types"
	"github.com/influenzanet/go-utils/pkg/constants"

	messageAPI "github.com/influenzanet/messaging-service/pkg/api/messaging_service"
	umAPI "github.com/influenzanet/user-management-service/pkg/api"
)

type InvitationResendReq struct {
	AccountID     []string `json:"accounts"`
	InstanceID    string   `json:"instance"`
	TokenLifetime int64    `json:"lifetime"`
}

type InvitationResendResult struct {
	State string `json:"state"`
	Error string `json:"error"`
}

type InvitationResendResponse struct {
	Scanned int                               `json:"scanned"`
	Results map[string]InvitationResendResult `json:"results"`
}

func (h *HttpEndpoints) InvitationResendHandl(c *gin.Context) {
	//token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req InvitationResendReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.AccountID) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AccountID cannot be empty"})
		return
	}

	accounts := make(map[string]interface{}, len(req.AccountID))
	for _, id := range req.AccountID {
		accounts[id] = nil
	}

	instanceID := req.InstanceID

	tokenLifeTime := req.TokenLifetime

	if tokenLifeTime <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TokenLifetime must be greater than 0"})
		return
	}

	filters := &umAPI.StreamUsersMsg_Filters{
		OnlyConfirmedAccounts:    false,
		UseReminderWeekdayFilter: false,
	}

	r := InvitationResendResponse{
		Results: make(map[string]InvitationResendResult),
	}

	ctx := context.Background()

	stream, err := h.clients.UserManagement.StreamUsers(ctx,
		&umAPI.StreamUsersMsg{
			InstanceId: instanceID,
			Filters:    filters,
		},
	)
	if err != nil {
		logger.Error.Printf("%v", err)
		return
	}
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error.Printf("%v", err)
			break
		}
		r.Scanned += 1
		accountId := user.Account.AccountId
		//fmt.Printf("Account %s", accountId)
		if _, ok := accounts[accountId]; !ok {
			continue
		}
		if user.Account.AccountConfirmedAt > 0 {
			r.Results[accountId] = InvitationResendResult{State: "active"}
			continue
		}
		err = h.sendInvitation(ctx, user, instanceID, tokenLifeTime)
		if err != nil {
			r.Results[accountId] = InvitationResendResult{State: "error", Error: err.Error()}
		} else {
			r.Results[accountId] = InvitationResendResult{State: "sent"}
		}
	}
	c.JSON(200, &r)
}

func (h *HttpEndpoints) sendInvitation(ctx context.Context, user *umAPI.User, instanceID string, expiresIn int64) error {

	userId := user.Id
	accountID := user.Account.AccountId
	resp, err := h.clients.UserManagement.GenerateTempToken(ctx, &api_types.TempTokenInfo{
		UserId:     userId,
		Purpose:    constants.TOKEN_PURPOSE_INVITATION,
		InstanceId: instanceID,
		Expiration: time.Now().Unix() + expiresIn,
		Info: map[string]string{
			"type":  "email",
			"email": accountID,
		},
	})

	if err != nil {
		return err
	}

	tempToken := resp.Token

	// ---> Trigger message sending
	_, err = h.clients.MessagingService.SendInstantEmail(ctx, &messageAPI.SendEmailReq{
		InstanceId:  instanceID,
		To:          []string{accountID},
		MessageType: constants.EMAIL_TYPE_INVITATION,
		ContentInfos: map[string]string{
			"token": tempToken,
		},
		PreferredLanguage: user.Account.PreferredLanguage,
		UseLowPrio:        true,
	})
	return err
}
