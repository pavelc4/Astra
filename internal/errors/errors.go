package errors

type ErrorCode string

const (
	CodeMissingParam  ErrorCode = "MISSING_PARAMETER"
	CodeInvalidURL    ErrorCode = "INVALID_URL"
	CodeUpstreamError ErrorCode = "UPSTREAM_FAILED"
	CodeNotFound      ErrorCode = "NOT_FOUND"
	CodeInternalError ErrorCode = "INTERNAL_ERROR"
)

type AppError struct {
	Message string
	Code    ErrorCode
	Status  int
}

func (e *AppError) Error() string {
	return e.Message
}

func NewDefault(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeInternalError, Status: 500}
}

func NewValidation(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeMissingParam, Status: 400}
}

func NewInvalidURL(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeInvalidURL, Status: 422}
}

func NewUpstream(msg string) *AppError {
	return &AppError{Message: msg, Code: CodeUpstreamError, Status: 502}
}
