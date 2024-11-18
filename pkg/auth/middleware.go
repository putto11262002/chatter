package auth

import (
	"context"
)

type sessionKey = string

const key sessionKey = "session"

func ContextWithSession(ctx context.Context, session Session) context.Context {
	return context.WithValue(ctx, key, session)
}

func SessionFromContext(ctx context.Context) (Session, bool) {
	session, ok := ctx.Value(key).(Session)
	return session, ok
}
