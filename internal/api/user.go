package api

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"example.com/go-chat/pkg/auth"
	"example.com/go-chat/pkg/user"
)

const (
	AuthCookieName = "auth_token"
)

type UserHandler struct {
	userStore user.UserStore
	auth      auth.Auth
}

func NewUserHandler(userStore user.UserStore, auth auth.Auth) *UserHandler {
	return &UserHandler{userStore: userStore, auth: auth}
}

type SignupPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type SigninPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SigninResponse struct {
	Token    string    `json:"token"`
	ExpireAt time.Time `json:"expireAt"`
}

type UserResponse struct {
	Username string `json:"username"`
	Name     string `json:"name"`
}

func (h *UserHandler) SignupHandler(w http.ResponseWriter, r *http.Request) error {
	var payload SignupPayload

	if err := DecodeJson(r.Body, &payload); err != nil {
		return err
	}

	defer r.Body.Close()

	input := user.User{
		Username: payload.Username,
		Password: payload.Password,
		Name:     payload.Name,
	}

	if err := h.userStore.CreateUser(r.Context(), input); err != nil {
		if errors.Is(err, user.ErrConflictedUser) {
			return NewApiError(err.Error(), http.StatusConflict)
		}

		return err
	}

	w.WriteHeader(http.StatusCreated)

	return nil
}

func (h *UserHandler) SigninHandler(w http.ResponseWriter, r *http.Request) error {
	var payload SigninPayload
	DecodeJson(r.Body, &payload)
	defer r.Body.Close()

	token, exp, err := h.auth.NewSession(r.Context(), payload.Username, payload.Password)

	if err != nil {
		if errors.Is(err, auth.ErrBadCredentials) {
			return NewApiError(err.Error(), http.StatusUnauthorized)
		}
		return err
	}

	cookie := http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		Expires:  exp,
		HttpOnly: true,
		Path:     "/",
	}

	http.SetCookie(w, &cookie)

	WriteJsonResponse(w, SigninResponse{Token: token, ExpireAt: exp})

	return nil
}

func (h *UserHandler) SignoutHandler(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (h *UserHandler) MeHandler(w http.ResponseWriter, r *http.Request) error {
	session := sessionFromRequest(r)
	user, err := h.userStore.GetUserByUsername(r.Context(), session.Username)
	if err != nil {
		return fmt.Errorf("get user by username: %w", err)
	}

	if user == nil {
		return NewApiError("unauthenticated", http.StatusUnauthorized)
	}

	WriteJsonResponse(w, UserResponse{Username: user.Username, Name: user.Name})
	return nil
}

func (h *UserHandler) GetUserByIDHandler(w http.ResponseWriter, r *http.Request) error {
	user, err := h.userStore.GetUserByUsername(r.Context(), r.PathValue("userID"))
	if err != nil {
		return err
	}

	if user == nil {
		return NewApiError("user not found", http.StatusNotFound)
	}

	WriteJsonResponse(w, UserResponse{Username: user.Username, Name: user.Name})
	return nil
}

// sessionFromRequest extracts the session from the request context.
// It must be called in handlers that are protected by the JWTMiddleware.
// It panics if the session is not found in the request context.
func sessionFromRequest(r *http.Request) auth.Session {
	session, ok := auth.SessionFromContext(r.Context())
	if !ok {
		panic("session not found in request context: call this function in handlers that are protected by JWTMiddleware")
	}
	return session
}

// JWTMiddleware extracts the JWT token from the request and validates it and attaches the session to the request context.
// The session is gaurenteed to be attached to the request context if the JWT token is valid for subsequent handlers.
func JWTMiddleware(_auth auth.Auth) ApiMiddleware {

	return func(next http.Handler) ApiHandleFunc {

		authErr := NewApiError("Unauthenticated", http.StatusUnauthorized)

		return ApiHandleFunc((func(w http.ResponseWriter, r *http.Request) error {
			ctx := r.Context()

			cookie, err := r.Cookie(AuthCookieName)
			if err != nil {
				return authErr
			}

			if cookie == nil {
				return authErr
			}

			if cookie.Valid() != nil {
				return authErr
			}

			session, err := _auth.Session(ctx, cookie.Value)

			if err != nil {
				if errors.Is(err, auth.ErrUnauthenticated) {
					return authErr
				}
				return err
			}

			newCtx := auth.ContextWithSession(ctx, *session)

			next.ServeHTTP(w, r.WithContext(newCtx))

			return nil

		}))
	}
}

// MeMiddleware is a middleware that replaces the path parameter with the username of the authenticated user if the path parameter is "me".
func MeMiddleware(pathParam string, matcher *regexp.Regexp) ApiMiddleware {

	return func(next http.Handler) ApiHandleFunc {

		return ApiHandleFunc((func(w http.ResponseWriter, r *http.Request) error {
			session := sessionFromRequest(r)

			v := r.PathValue(pathParam)
			if matcher.Match([]byte(v)) {

				r.SetPathValue(pathParam, session.Username)
			}

			next.ServeHTTP(w, r)

			return nil

		}))
	}
}
