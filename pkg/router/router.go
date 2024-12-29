package router

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"runtime"

	"github.com/go-chi/chi/v5"
)

var DefaultError = JsonError{
	Code: http.StatusInternalServerError,
	Err:  "internal server error",
}

// Router is a wrapper around chi.Router that provides error handling.
// Handlers can return an error that will then get mapped to an error response.
// Error mappers can be registers for specific error types to provide custom error responses.
type Router struct {
	chi.Router
	errorMappers map[string]ErrorMapper
	defaultError JsonError
	logger       *slog.Logger
}

func New(opts ...RouterOption) *Router {
	return new(chi.NewRouter(), opts...)
}

type RouterOption func(*Router)

func WithLogger(logger *slog.Logger) RouterOption {
	return func(r *Router) {
		r.logger = logger
	}
}

func WithDefaultError(err JsonError) RouterOption {
	return func(r *Router) {
		r.defaultError = err
	}
}

func new(chiRouter chi.Router, opts ...RouterOption) *Router {
	router := &Router{
		Router:       chiRouter,
		errorMappers: make(map[string]ErrorMapper),
		defaultError: DefaultError,
		logger:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	for _, opt := range opts {
		opt(router)
	}
	return router
}

// HandlerFunc is a function that handles an HTTP request and returns an error.
// When the handler fails to handler to request it should not write anything to the response writer
// instead it should return an error that will be mapped to an error response.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

type Middleware func(http.Handler) HandlerFunc

// ErrorMapper is a function that maps go errors to API errors.
type ErrorMapper func(error) Error

func (a *Router) RegisterErrorMapper(err error, fn ErrorMapper) {
	a.errorMappers[err.Error()] = fn
}

// mapError maps a go error to an API error.
// The mapping works as following:
//   - if the error is already an APIError it will be returned as is.
//   - if the error is a non-api error it will be mapped using the error mappers.
//   - if no error mapper is found the default error will be returned.
func (a *Router) mapError(err error) Error {
	apiErr, ok := err.(JsonError)
	if ok {
		return apiErr
	}

	fn, ok := a.errorMappers[err.Error()]
	if !ok {
		return a.defaultError
	}
	return fn(err)
}

func (a *Router) handleWithErr(h HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err != nil {
			handlerFn := runtime.FuncForPC(reflect.ValueOf(h).Pointer())
			a.logger.Error(err.Error(), slog.String("handler", handlerFn.Name()))
			resError := a.mapError(err)
			w.WriteHeader(resError.StatusCode())
			if err := json.NewEncoder(w).Encode(resError); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
	}
}

func (a *Router) Get(path string, h HandlerFunc) {
	a.Router.Get(path, a.handleWithErr(h))
}

func (a *Router) Post(path string, h HandlerFunc) {
	a.Router.Post(path, a.handleWithErr(h))
}

func (a *Router) Put(path string, h HandlerFunc) {
	a.Router.Put(path, a.handleWithErr(h))
}

func (a *Router) Delete(path string, h HandlerFunc) {
	a.Router.Delete(path, a.handleWithErr(h))
}

func (a *Router) Route(path string, f func(r *Router)) {
	a.Router.Route(path, func(r chi.Router) {
		f(new(r))
	})
}

func (a *Router) Group(f func(r *Router)) *Router {
	ch := a.Router.Group(func(r chi.Router) {
		f(&Router{Router: r})
	})
	return new(ch)
}

func (a *Router) Use(middleware Middleware) {
	a.Router.Use(func(h http.Handler) http.Handler {
		return a.handleWithErr(middleware(h))
	})
}

func (a *Router) With(middleware Middleware) *Router {
	ch := a.Router.With(func(h http.Handler) http.Handler {
		return a.handleWithErr(middleware(h))
	})
	return new(ch)
}
