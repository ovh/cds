package sdk

import "context"

type localContextKey int

const (
	contextLocalServiceName localContextKey = iota
	contextLocalServiceType
)

// ContextWithLocalService injects a local service identity into the context.
// This is used by the in-process transport (LocalRoundTripper) to identify the calling
// service without JWT tokens. Only in-process calls can set these values — external
// HTTP requests cannot forge them.
func ContextWithLocalService(ctx context.Context, serviceName, serviceType string) context.Context {
	ctx = context.WithValue(ctx, contextLocalServiceName, serviceName)
	ctx = context.WithValue(ctx, contextLocalServiceType, serviceType)
	return ctx
}

// LocalServiceFromContext extracts the local service identity from the context.
// Returns empty strings and false if the context does not contain local service info.
func LocalServiceFromContext(ctx context.Context) (serviceName string, serviceType string, ok bool) {
	name, nameOk := ctx.Value(contextLocalServiceName).(string)
	sType, typeOk := ctx.Value(contextLocalServiceType).(string)
	if !nameOk || !typeOk || name == "" {
		return "", "", false
	}
	return name, sType, true
}
