package openapi

type BadRequest struct {
	Error BadRequestError `json:"error" validate:"required"`
}

type BadRequestError struct {
	Code    string `json:"code" validate:"required" example:"invalid_request"`
	Message string `json:"message,omitempty"`
}

type InvalidAPIKey struct {
	Error InvalidAPIKeyError `json:"error" validate:"required"`
}

type InvalidAPIKeyError struct {
	Code    string `json:"code" validate:"required" example:"invalid_api_key"`
	Message string `json:"message,omitempty" example:"Provide an active project environment API key."`
}

type NotAuthenticated struct {
	Error NotAuthenticatedError `json:"error" validate:"required"`
}

type NotAuthenticatedError struct {
	Code    string `json:"code" validate:"required" example:"not_authenticated"`
	Message string `json:"message,omitempty"`
}

type RateLimited struct {
	Error RateLimitedError `json:"error" validate:"required"`
}

type RateLimitedError struct {
	Code    string `json:"code" validate:"required" example:"too_many_attempts"`
	Message string `json:"message,omitempty"`
}

type ServerError struct {
	Error ServerErrorError `json:"error" validate:"required"`
}

type ServerErrorError struct {
	Code    string `json:"code" validate:"required" example:"authentication_failed"`
	Message string `json:"message,omitempty"`
}
