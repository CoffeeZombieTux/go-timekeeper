package logger

const (
	LogMessageOutgoingHTTPResponse            = "outgoing_http_response"
	LogMessageFailedToInitializeDatabase      = "Failed to initialize database"
	LogMessageServerFailed                    = "Server failed"
	LogMessageGracefulServerShutdownFailed    = "Graceful server shutdown failed"
	LogMessageFailedToCloseDatabaseConnection = "Failed to close database connection"
	LogMessageDatabaseHealthCheckFailed       = "Database health check failed"
	LogMessageFailedToCloseRows               = "failed_to_close_rows"
	LogMessageFailedToCloseResponseBody       = "failed_to_close_response_body"
	LogMessageFailedToRegisterUser            = "failed_to_register_user"
	LogMessageFailedToLoginUser               = "failed_to_login_user"
	LogMessageFailedToLogoutUser              = "failed_to_logout_user"
	LogMessageFailedToChangePassword          = "failed_to_change_password"
	LogMessageFailedToGetUser                 = "failed_to_get_user"
	LogMessageFailedToDeleteUser              = "failed_to_delete_user"
	LogMessageFailedToRefreshUserToken        = "failed_to_refresh_user_token"
	LogMessageFailedToCreateProject           = "failed_to_create_project"
	LogMessageFailedToUpdateProject           = "failed_to_update_project"
	LogMessageFailedToDeleteProject           = "failed_to_delete_project"
	LogMessageFailedToGetProject              = "failed_to_get_project"
)
