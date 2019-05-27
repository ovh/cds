package api

import (
	"context"
)

type contextKey int

const (
	contextUserAuthentified contextKey = iota
	contextProvider
	contextAPIConsumer
	contextJWT
	contextJWTRaw
	contextScope
	contextWorkflowTemplate
)

// ContextValues retuns auth values of a context
func ContextValues(ctx context.Context) map[interface{}]interface{} {
	return map[interface{}]interface{}{
		//contextHatchery: ctx.Value(contextHatchery),
		//contextService:  ctx.Value(contextService),
		//contextWorker:   ctx.Value(contextWorker),
		//contextUser:     ctx.Value(contextUser),
	}
}
