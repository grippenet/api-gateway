package v1

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/coneno/logger"
	"github.com/crewjam/saml/samlsp"
	"github.com/gin-gonic/gin"
	"github.com/influenzanet/api-gateway/pkg/utils"
	"github.com/influenzanet/go-utils/pkg/api_types"
	"github.com/influenzanet/go-utils/pkg/constants"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"

	studyAPI "github.com/influenzanet/study-service/pkg/api"
	umAPI "github.com/influenzanet/user-management-service/pkg/api"
)

func (h *HttpEndpoints) loginWithEmailAsParticipantHandl(c *gin.Context) {
	var req umAPI.LoginWithEmailMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.AsParticipant = true

	token, err := h.clients.UserManagement.LoginWithEmail(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, token)
}

func (h *HttpEndpoints) resendVerificationCodeHandl(c *gin.Context) {
	var req umAPI.SendVerificationCodeReq
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.clients.UserManagement.SendVerificationCode(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) getVerificationCodeWithTokenHandl(c *gin.Context) {
	var req umAPI.AutoValidateReq
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.clients.UserManagement.AutoValidateTempToken(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) loginWithEmailForManagementHandl(c *gin.Context) {
	var req umAPI.LoginWithEmailMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.AsParticipant = false

	resp, err := h.clients.UserManagement.LoginWithEmail(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) signupWithEmailHandl(c *gin.Context) {
	var req umAPI.SignupWithEmailMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.clients.UserManagement.SignupWithEmail(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) signupWithEmailHandlV2(c *gin.Context) {
	var req umAPI.SignupWithEmailMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.clients.UserManagement.SignupWithEmail(context.Background(), &req)
	h.handleGRPCResponse(c, resp, err)
}

type customHandlerMethod func(*gin.Context) (protoreflect.ProtoMessage, error)

func (h *HttpEndpoints) signupWithEmailHandlV3(c *gin.Context) {
	h.grpcCallHandler(
		c,
		func(c *gin.Context) (protoreflect.ProtoMessage, error) {
			var req umAPI.SignupWithEmailMsg
			if err := h.JsonToProto(c, &req); err != nil {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}

			if len(req.InfoCheck) > 0 {
				req.Password = ""
				logger.Error.Printf("honeypot value filled with request: %v", &req)
				return nil, status.Error(codes.InvalidArgument, "invalid request")
			}

			return h.clients.UserManagement.SignupWithEmail(context.Background(), &req)
		},
	)
}

func (h *HttpEndpoints) grpcCallHandler(c *gin.Context, customMethod customHandlerMethod) {
	resp, err := customMethod(c)
	h.handleGRPCResponse(c, resp, err)
}

func (h *HttpEndpoints) handleGRPCResponse(c *gin.Context, resp protoreflect.ProtoMessage, err error) {
	if err != nil {
		st := status.Convert(err)
		logger.Error.Println(st.Message())
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) tokenRenewHandl(c *gin.Context) {
	var req umAPI.RefreshJWTRequest
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.AccessToken = c.MustGet("encodedToken").(string)
	token, err := h.clients.UserManagement.RenewJWT(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, token)
}

func (h *HttpEndpoints) userPasswordChangeHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.PasswordChangeMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token

	resp, err := h.clients.UserManagement.ChangePassword(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		logger.Error.Println(st.Message())
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) changeAccountEmailHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.EmailChangeMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.ChangeAccountIDEmail(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) userSetPreferredLanguageHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.LanguageChangeMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.ChangePreferredLanguage(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) getUserHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	userID := c.Param("id")

	userRefReq := &umAPI.UserReference{
		Token:  token,
		UserId: userID,
	}

	resp, err := h.clients.UserManagement.GetUser(context.Background(), userRefReq)
	if err != nil {
		st := status.Convert(err)
		logger.Error.Println(st.Message())
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) deleteAccountHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.UserReference
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token

	// for all profiles, notify the study service that the account is deleted:
	userProfileIDs := []string{token.ProfilId}
	userProfileIDs = append(userProfileIDs, token.OtherProfileIds...)

	for _, profileId := range userProfileIDs {
		token.ProfilId = profileId
		// Notify study service that profile is deleted:
		if _, err := h.clients.StudyService.ProfileDeleted(context.Background(),
			token,
		); err != nil {
			logger.Error.Println(err)
		}
	}

	// delete account:
	resp, err := h.clients.UserManagement.DeleteAccount(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		logger.Error.Println(st.Message())
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) saveProfileHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.ProfileRequest
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.SaveProfile(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) removeProfileHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.ProfileRequest
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.RemoveProfile(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}

	token.ProfilId = req.Profile.Id
	// Notify study service that profile is deleted:
	if _, err := h.clients.StudyService.ProfileDeleted(context.Background(),
		token,
	); err != nil {
		logger.Error.Println(err)
	}
	logger.Debug.Println("profile and its confidential data is deleted.")

	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) userUpdateContactPreferencesHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.ContactPreferencesMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.UpdateContactPreferences(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) userAddEmailHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.ContactInfoMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.AddEmail(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) userRemoveEmailHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.ContactInfoMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.RemoveEmail(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) createUserHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.CreateUserReq
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.CreateUser(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

type MigrateUserReq struct {
	AccountID          string             `json:"accountId"`
	OldParticipantIDs  []string           `json:"oldParticipantIDs"` // per profile
	ProfileNames       []string           `json:"profileNames"`      // per profile
	InitialPassword    string             `json:"initialPassword"`
	PreferredLanguage  string             `json:"preferredLanguage"`
	Studies            []string           `json:"studies"`
	Use2FA             bool               `json:"use2FA"`
	AccountConfirmedAt int64              `json:"accountConfirmedAt"` // if account should not be removed automatically
	CreatedAt          int64              `json:"createdAt"`          // to override when account was created
	Reports            []*studyAPI.Report `json:"reports"`            // create these reports for the user as well
}

func (h *HttpEndpoints) migrateUserHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req MigrateUserReq
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.ProfileNames) > len(req.OldParticipantIDs) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "request data must have the same number of entries for oldParticipantIDs and profileNames, if profileNames are not empty."})
		return
	}

	// Create user
	cuReq := umAPI.CreateUserReq{
		AccountId:          req.AccountID,
		InitialPassword:    req.InitialPassword,
		PreferredLanguage:  req.PreferredLanguage,
		Roles:              []string{constants.USER_ROLE_PARTICIPANT},
		Token:              token,
		Use_2Fa:            req.Use2FA,
		ProfileNames:       req.ProfileNames,
		AccountConfirmedAt: req.AccountConfirmedAt,
		CreatedAt:          req.CreatedAt,
	}
	newUser, err := h.clients.UserManagement.CreateUser(context.Background(), &cuReq)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}

	for _, studyKey := range req.Studies {
		for pIndex, profile := range newUser.Profiles {
			newUserToken := &api_types.TokenInfos{
				Id:               newUser.Id,
				AccountConfirmed: true,
				InstanceId:       token.InstanceId,
				ProfilId:         profile.Id,
			}

			// enter studies:
			_, err = h.clients.StudyService.EnterStudy(context.TODO(), &studyAPI.EnterStudyRequest{
				Token:     newUserToken,
				StudyKey:  studyKey,
				ProfileId: profile.Id,
			})
			if err != nil {
				st := status.Convert(err)
				c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
				return
			}

			// To improve privace, reduce resoltuion of the timestamp
			noon := time.Now().Truncate(24 * time.Hour).Add(12 * time.Hour).Unix()

			// submit migration survey
			_, err = h.clients.StudyService.SubmitResponse(context.TODO(), &studyAPI.SubmitResponseReq{
				Token:     newUserToken,
				StudyKey:  studyKey,
				ProfileId: profile.Id,
				Response: &studyAPI.SurveyResponse{
					Key:         "migration",
					SubmittedAt: noon,
					Responses: []*studyAPI.SurveyItemResponse{
						{Key: "migration.OldID", Response: &studyAPI.ResponseItem{Key: "rg", Items: []*studyAPI.ResponseItem{
							{Key: "ic", Value: req.OldParticipantIDs[pIndex]},
						}}},
					},
					Context: map[string]string{
						"engineVersion": "-",
					},
				},
			})
			if err != nil {
				st := status.Convert(err)
				c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
				return
			}

			// Create reports:
			for _, report := range req.Reports {
				logger.Debug.Println(report)
				if report.ProfileId != req.OldParticipantIDs[pIndex] {
					continue
				}
				_, err = h.clients.StudyService.CreateReport(context.TODO(), &studyAPI.CreateReportReq{
					Token:     newUserToken,
					StudyKey:  report.StudyKey,
					ProfileId: profile.Id,
					Report:    report,
				})
				if err != nil {
					logger.Error.Printf("unexpected error: %v", err.Error())
				}
			}
		}
	}

	h.SendProtoAsJSON(c, http.StatusOK, newUser)
}

func (h *HttpEndpoints) findNonParticipantUsersHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.FindNonParticipantUsersMsg
	req.Token = token
	resp, err := h.clients.UserManagement.FindNonParticipantUsers(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) userAddRoleHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.RoleMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.AddRoleForUser(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) userRemoveRoleHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.RoleMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.RemoveRoleForUser(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) revokeRefreshTokensHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)

	var req umAPI.RevokeRefreshTokensReq
	req.Token = token
	resp, err := h.clients.UserManagement.RevokeAllRefreshTokens(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) initiatePasswordResetHandl(c *gin.Context) {
	var req umAPI.InitiateResetPasswordMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.clients.UserManagement.InitiatePasswordReset(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) getInfosForPasswordResetHandl(c *gin.Context) {
	var req umAPI.GetInfosForResetPasswordMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.clients.UserManagement.GetInfosForPasswordReset(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) passwordResetHandl(c *gin.Context) {
	var req umAPI.ResetPasswordMsg
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.clients.UserManagement.ResetPassword(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) verifyUserContactHandl(c *gin.Context) {
	var req umAPI.TempToken
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.clients.UserManagement.VerifyContact(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) resendContanctVerificationEmailHandl(c *gin.Context) {
	token := c.MustGet("validatedToken").(*api_types.TokenInfos)
	var req umAPI.ResendContactVerificationReq
	if err := h.JsonToProto(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Token = token
	resp, err := h.clients.UserManagement.ResendContactVerification(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

func (h *HttpEndpoints) unsubscribeNewsletterHandl(c *gin.Context) {
	var req umAPI.TempToken
	req.Token = c.DefaultQuery("token", "")
	resp, err := h.clients.UserManagement.UseUnsubscribeToken(context.Background(), &req)
	if err != nil {
		st := status.Convert(err)
		c.JSON(utils.GRPCStatusToHTTP(st.Code()), gin.H{"error": st.Message()})
		return
	}
	h.SendProtoAsJSON(c, http.StatusOK, resp)
}

type GroupInfo struct {
	Customer   string
	Prefix     string
	InstanceID string
	Role       string
}

func parseSAMLgroupInfo(groups []string) []GroupInfo {
	sep := "-"
	infos := []GroupInfo{}
	for _, groupText := range groups {
		parts := strings.Split(groupText, sep)
		if len(parts) != 4 {
			logger.Error.Printf("'%s' has %d parts when using '%s' as a separator but 4 are expected.", groupText, len(parts), sep)
			continue
		}

		c := GroupInfo{
			Customer:   parts[0],
			Prefix:     parts[1],
			InstanceID: parts[2],
			Role:       parts[3],
		}
		infos = append(infos, c)
	}
	return infos
}

func checkPermission(samlGroupInfos []GroupInfo, instanceID string, role string) (bool, *GroupInfo) {
	for _, g := range samlGroupInfos {
		if strings.EqualFold(g.InstanceID, instanceID) && strings.EqualFold(role, g.Role) {
			return true, &g
		}
	}
	return false, nil
}

type SAMLLoginInfo struct {
	Username   string
	Tokens     string
	InstanceID string
	Role       string
}

func (h *HttpEndpoints) loginWithSAML(w http.ResponseWriter, r *http.Request) {
	instanceIDs, ok := r.URL.Query()["instance"]
	if !ok || len(instanceIDs[0]) < 1 {
		http.Error(w, "Url Param 'instance' is missing", http.StatusBadRequest)
		return
	}
	roles, ok := r.URL.Query()["role"]
	if !ok || len(roles[0]) < 1 {
		http.Error(w, "Url Param 'role' is missing", http.StatusBadRequest)
		return
	}

	instanceID := instanceIDs[0]
	role := roles[0]

	s := samlsp.SessionFromContext(r.Context())
	if s == nil {
		logger.Error.Println("session not found")
		return
	}

	jwtSessionClaims, ok := s.(samlsp.JWTSessionClaims)
	if !ok {
		logger.Error.Println("Unable to decode session into JWTSessionClaims")
		return
	}

	email := jwtSessionClaims.Subject

	sa, ok := s.(samlsp.SessionWithAttributes)
	if !ok {
		logger.Error.Println("attributes not found")
		return
	}

	attributes := sa.GetAttributes()
	groups, ok := attributes["http://schemas.xmlsoap.org/claims/Group"]
	if !ok {
		err := fmt.Errorf("group infos not found in the response token for %s", email)
		logger.Error.Println(err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	groupInfos := parseSAMLgroupInfo(groups)

	hasPermission, usedGroupInfo := checkPermission(groupInfos, instanceID, role)
	if !hasPermission {
		err := fmt.Errorf("'%s' is not authorized to access '%s' with role '%s'.", email, instanceID, role)
		logger.Error.Println(err.Error())
		logger.Debug.Printf("valid group infos are %v", groupInfos)
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	req := umAPI.LoginWithExternalIDPMsg{
		InstanceId: instanceID,
		Email:      email,
		Role:       strings.ToUpper(role),
		Customer:   usedGroupInfo.Customer,
		GroupInfo:  strings.Join(groups, ";"),
		Idp:        h.samlConfig.IDPUrl,
	}

	resp, err := h.clients.UserManagement.LoginWithExternalIDP(context.Background(), &req)
	if err != nil {
		logger.Error.Println(err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	loginInfos := SAMLLoginInfo{
		Username:   email,
		InstanceID: instanceID,
		Role:       role,
		Tokens: strings.Join([]string{
			resp.Token.AccessToken,
			resp.Token.RefreshToken,
		}, "<!>"),
	}

	parsedTemplate, _ := template.ParseFiles(h.samlConfig.TemplatePathLoginSuccess)
	err = parsedTemplate.Execute(w, loginInfos)
	if err != nil {
		logger.Error.Println(err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	if err != nil {
		logger.Error.Println("Error executing template :", err)
		return
	}

	// c.Data(http.StatusOK, "text/html; charset=utf-8", tpl.Bytes())
	//fmt.Fprintf(w, "Logged in as: %s, Token contents, %+v!\n\n%v \n\n %s - %s \n\n%s", email, sa.GetAttributes(), groupInfos, instanceID, role, resp.Token.AccessToken)
}
