package api

type contextKey int

const (
	contextSession contextKey = iota
	contextAPIConsumer
	contextDate
)
