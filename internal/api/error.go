package api

type ApiError[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func NewApiError(message string, code int) *ApiError[interface{}] {
	return &ApiError[interface{}]{
		Code:    code,
		Message: message,
	}
}

func NewApiErrorWithData[T any](message string, code int, data T) *ApiError[T] {
	return &ApiError[T]{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func (e ApiError[T]) Error() string {
	return e.Message
}
