package api

type contextKey int

const (
	contextClaims contextKey = iota
	contextSession
	contextUserConsumer
	contextDriverManifest
	contextDate
)
