package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func callbackServer(ctx context.Context, t *testing.T, out chan http.Request) {
	srv := &http.Server{Addr: ":8081"}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		out <- *r
		io.WriteString(w, "Yeah !\n")
		fmt.Println("Handler")
	})

	go func() {
		fmt.Println("Starting server")
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			t.Logf("Httpserver: ListenAndServe() error: %s", err)
		}
		close(out)
	}()

	<-ctx.Done()
	fmt.Println("Stopping server")
	srv.Shutdown(ctx)
}
