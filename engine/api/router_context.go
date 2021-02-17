package api

type contextKey int

const (
	contextClaims contextKey = iota
	contextSession
	contextAPIConsumer
	contextDate
)
