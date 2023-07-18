package api

type contextKey int

const (
	contextClaims contextKey = iota
	contextSession
	contextUserConsumer
	contextHatcheryConsumer
	contextDriverManifest
	contextDate
	contextWorker
)
