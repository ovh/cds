package api

type contextKey int

const (
	contextClaims contextKey = iota
	contextSession
	contextConsumer
	contextDriverManifest
	contextDate
)
