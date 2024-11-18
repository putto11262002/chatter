package api

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ApiMux struct {
	chi.Router
}

func NewAPiRouter() *ApiMux {
	return &ApiMux{
		Router: chi.NewRouter(),
	}
}

type ApiHandleFunc func(http.ResponseWriter, *http.Request) error

func (h ApiHandleFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	err := h(w, r)

	if err == nil {
		return
	}

	if apiErr, ok := err.(*ApiError[interface{}]); ok {

		if err := WriteJsonResponseWithStatusCode(w, apiErr, apiErr.Code); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		return
	}

	log.Printf("Internal Server Error: %v", err)

	apiErr := NewApiError("Internal Server Error", http.StatusInternalServerError)

	if err := WriteJsonResponseWithStatusCode(w, apiErr, apiErr.Code); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

}

type ApiMiddleware func(http.Handler) ApiHandleFunc

func (a *ApiMux) Get(path string, h ApiHandleFunc) {
	a.Router.Get(path, h.ServeHTTP)
}

func (a *ApiMux) Post(path string, h ApiHandleFunc) {
	a.Router.Post(path, h.ServeHTTP)
}

func (a *ApiMux) Put(path string, h ApiHandleFunc) {
	a.Router.Put(path, h.ServeHTTP)
}

func (a *ApiMux) Delete(path string, h ApiHandleFunc) {
	a.Router.Delete(path, h.ServeHTTP)
}

func (a *ApiMux) Route(path string, f func(r *ApiMux)) {
	a.Router.Route(path, func(r chi.Router) {
		f(&ApiMux{Router: r})
	})

}

func (a *ApiMux) Use(middleware ApiMiddleware) {
	a.Router.Use(func(h http.Handler) http.Handler {
		return middleware(h)
	})

}

func (a *ApiMux) With(middleware ApiMiddleware) *ApiMux {
	ch := a.Router.With(func(h http.Handler) http.Handler {
		return middleware(h)
	})
	return &ApiMux{Router: ch}
}
