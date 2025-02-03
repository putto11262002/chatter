package core

import (
	"context"
	"errors"
	"net/http"

	"github.com/putto11262002/chatter/pkg/router"
)

const (
	key            sessionKey = "session"
	AuthCookieName            = "auth_token"
)

type sessionKey = string

func contextWithSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, key, session)
}

func sessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(key).(Session)
	return session, ok
}

// SessionFromRequest extracts the session from the request context.
// It must be called in handlers that are protected by the JWTMiddleware.
// It panics if the session is not found in the request context.
func SessionFromRequest(r *http.Request) Session {
	session, ok := sessionFromContext(r.Context())
	if !ok {
		panic("session not found in request context: call this function in handlers that are protected by JWTMiddleware")
	}
	return session
}

func CookieFromRequest(session Session, httpOnly bool, path string) *http.Cookie {
	return &http.Cookie{
		Name:     AuthCookieName,
		Value:    session.Token,
		Expires:  session.ExpiresAt,
		HttpOnly: httpOnly,
		Path:     path,
	}
}

// JWTMiddleware extracts the JWT token from the request and validates it and attaches the session to the request context.
// The session is gaurenteed to be attached to the request context if the JWT token is valid for subsequent handlers.
func JWTMiddleware(a AuthStore) router.Middleware {

	return func(next http.Handler) router.HandlerFunc {

		authErr := router.NewJsonError(http.StatusUnauthorized, "unauthenticated")

		return router.HandlerFunc((func(w http.ResponseWriter, r *http.Request) error {
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

			session, err := a.Session(ctx, cookie.Value)

			if err != nil {
				if errors.Is(err, ErrUnauthenticated) {
					return authErr
				}
				return err
			}

			newCtx := contextWithSession(ctx, *session)

			next.ServeHTTP(w, r.WithContext(newCtx))

			return nil

		}))
	}
}
