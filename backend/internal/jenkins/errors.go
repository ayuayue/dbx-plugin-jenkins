package jenkins

import "fmt"

// ErrorKind 是稳定的错误分类，UI 据此做本地化提示。
type ErrorKind string

const (
	ErrInvalidConfig     ErrorKind = "invalid_config"
	ErrReadOnly          ErrorKind = "read_only"
	ErrUnauthorized      ErrorKind = "unauthorized"
	ErrForbidden         ErrorKind = "forbidden"
	ErrNotFound          ErrorKind = "not_found"
	ErrRequest           ErrorKind = "request_failed"
	ErrRejected          ErrorKind = "rejected"
	ErrServer            ErrorKind = "server_error"
	ErrTimeout           ErrorKind = "timeout"
	ErrNetwork           ErrorKind = "network"
	ErrStagesUnavailable ErrorKind = "stages_unavailable"
)

// APIError 是 Jenkins 调用失败的统一错误。
type APIError struct {
	Kind       ErrorKind
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s (HTTP %d)", e.Message, e.StatusCode)
	}
	return e.Message
}
