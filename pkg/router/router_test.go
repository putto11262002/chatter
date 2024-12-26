package router

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ErrorMapper(t *testing.T) {
	router := New()

	tcs := []struct {
		err    error
		mapper ErrorMapper
		exp    JsonError
	}{
		{
			err: errors.New("custom error"),
			mapper: func(err error) JsonError {
				return JsonError{
					Code: 400,
					Err:  err.Error(),
				}
			},
			exp: JsonError{
				Code: 400,
				Err:  "custom error",
			},
		},
		{
			err:    errors.New("random error"),
			mapper: nil,
			exp:    router.defaultError,
		},
		{
			err: JsonError{
				Code: 400,
				Err:  "API Error",
			},
			mapper: nil,
			exp: JsonError{
				Code: 400,
				Err:  "API Error",
			},
		},
	}

	for _, tc := range tcs {
		if tc.mapper != nil {
			router.RegisterErrorMapper(tc.err, tc.mapper)
		}
		got := router.mapError(tc.err)
		assert.Equal(t, tc.exp, got)
	}
}
