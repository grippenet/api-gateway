package models

const (
	ENV_GRPC_MAX_MSG_SIZE = "GRPC_MAX_MSG_SIZE"

	ENV_REQUIRE_MUTUAL_TLS     = "REQUIRE_MUTUAL_TLS"
	ENV_MUTUAL_TLS_SERVER_CERT = "MUTUAL_TLS_SERVER_CERT"
	ENV_MUTUAL_TLS_SERVER_KEY  = "MUTUAL_TLS_SERVER_KEY"
	ENV_MUTUAL_TLS_CA_CERT     = "MUTUAL_TLS_CA_CERT"
)

const (
	DefaultGRPCMaxMsgSize = 4194304
)
