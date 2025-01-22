package core

type Error struct {
	msg string
	// sensitive is a flag to indicate if the error is sensitive or not.
	// If it is not, it can be returned to the client.
	Sensitive bool
}

func NewError(msg string, sensitive bool) *Error {
	return &Error{msg: msg, Sensitive: sensitive}
}

func NewErrorf(format string, args ...interface{}) *Error {
	return &Error{msg: format, Sensitive: false}
}

func NewSensitiveError(msg string) *Error {
	return &Error{msg: msg, Sensitive: true}
}

func NewInsensitiveError(msg string) *Error {
	return &Error{msg: msg, Sensitive: false}
}

func (e *Error) Error() string {
	return e.msg
}
