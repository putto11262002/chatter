package router

import (
	"encoding/json"
	"io"
)

type Error interface {
	error
	StatusCode() int
	Encode(w io.Writer) error
}

type JsonError struct {
	Code int    `json:"code"`
	Err  string `json:"error"`
}

func NewJsonError(code int, err string) JsonError {
	return JsonError{
		Code: code,
		Err:  err,
	}
}

func (e JsonError) StatusCode() int {
	return e.Code
}

func (e JsonError) Error() string {
	return e.Err
}

func (e JsonError) Encode(w io.Writer) error {
	return json.NewEncoder(w).Encode(e)
}
