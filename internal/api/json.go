package api

import (
	"encoding/json"
	"io"
	"net/http"
)

func DecodeJson(r io.Reader, v any) error {
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(v); err != nil {

		return err
	}

	return nil
}

func WriteJsonResponse(w http.ResponseWriter, v any) error {
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	err := encoder.Encode(v)
	if err != nil {
		return err
	}
	return nil
}

func WriteJsonResponseWithStatusCode(w http.ResponseWriter, v any, code int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	encoder := json.NewEncoder(w)
	err := encoder.Encode(v)
	if err != nil {
		return err
	}
	return nil
}
