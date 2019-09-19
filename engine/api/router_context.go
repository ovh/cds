package api

import (
	"context"
)

type contextKey int

const (
	contextSession contextKey = iota
	contextAPIConsumer
	contextJWT
	contextJWTRaw
	contextDate
	contextJWTFromCookie
)

// ContextValues retuns auth values of a context
func ContextValues(ctx context.Context) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		contextSession:       ctx.Value(contextSession),
		contextAPIConsumer:   ctx.Value(contextAPIConsumer),
		contextJWT:           ctx.Value(contextJWT),
		contextJWTRaw:        ctx.Value(contextJWTRaw),
		contextJWTFromCookie: ctx.Value(contextJWTFromCookie),
	}
}
