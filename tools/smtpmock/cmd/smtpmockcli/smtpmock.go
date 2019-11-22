package main

import (
	"context"

	"github.com/fsamin/smtp"
)

type Handler struct {
	pattern string
	handler smtp.HandlerFunc
}

func handle(pattern string, handler smtp.HandlerFunc) Handler {
	return Handler{pattern, handler}
}

func startServer(ctx context.Context, address string, handlers ...Handler) error {
	srv := smtp.NewServeMux()
	for i := range handlers {
		srv.HandleFunc(handlers[i].pattern, handlers[i].handler)
	}
	return smtp.ListenAndServeWithContext(ctx, address, srv)
}
