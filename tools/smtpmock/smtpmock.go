package smtpmock

import (
	"context"

	"github.com/fsamin/smtp"
)

type Handler struct {
	pattern string
	handler smtp.HandlerFunc
}

func Handle(pattern string, handler smtp.HandlerFunc) Handler {
	return Handler{pattern, handler}
}

func StartServer(ctx context.Context, address string, handlers ...Handler) error {
	srv := smtp.NewServeMux()
	for i := range handlers {
		srv.HandleFunc(handlers[i].pattern, handlers[i].handler)
	}
	return smtp.ListenAndServeWithContext(ctx, address, srv)
}
