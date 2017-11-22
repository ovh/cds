package api

// From https://golang.org/src/net/http/pprof/pprof.go
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/sdk/log"
)

// getProfileIndexHandler returns the profiles index
func (api *API) getProfileIndexHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		session := getUserSession(ctx)
		str := fmt.Sprintf(`<html>
			<head>
			<title>CDS Debug</title>
			</head>
			<body>
			CDS debug pprof<br>
			<br>
			profiles:<br>
			<table>
			{{range .}}
			<tr><td align=right>{{.Count}}<td><a href="debug/{{.Name}}?debug=1&session=%s">{{.Name}}</a>
			{{end}}
			</table>
			<br>
			<a href="debug/goroutine?debug=2&session=%s">Full goroutine stack dump</a><br>
			<a href="debug/trace?session=%s&seconds=5">Trace profile for 5 seconds</a><br>
			<a href="debug/cpu?session=%s&seconds=5">CPU profile for 5 seconds</a><br>
			</body>
			</html>
			`, session, session, session, session)

		var indexTmpl = template.Must(template.New("index").Parse(str))

		profiles := pprof.Profiles()
		if err := indexTmpl.Execute(w, profiles); err != nil {
			log.Error("getProfileIndexHandler> %v", err)
		}
		return nil
	}
}

// getProfileHandler responds with the pprof-formatted profile named by the request.
func (api *API) getProfileHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := mux.Vars(r)["name"]
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		debug, _ := strconv.Atoi(r.FormValue("debug"))
		p := pprof.Lookup(name)
		if p == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "Unknown profile: %s\n", name)
			return nil
		}
		gc, _ := strconv.Atoi(r.FormValue("gc"))
		if name == "heap" && gc > 0 {
			runtime.GC()
		}
		p.WriteTo(w, debug)
		return nil
	}
}

func sleep(w http.ResponseWriter, d time.Duration) {
	var clientGone <-chan bool
	if cn, ok := w.(http.CloseNotifier); ok {
		clientGone = cn.CloseNotify()
	}
	select {
	case <-time.After(d):
	case <-clientGone:
	}
}

func durationExceedsWriteTimeout(r *http.Request, seconds float64) bool {
	srv, ok := r.Context().Value(http.ServerContextKey).(*http.Server)
	return ok && srv.WriteTimeout != 0 && seconds >= srv.WriteTimeout.Seconds()
}

// getTraceHandler responds with the execution trace in binary form.
// Tracing lasts for duration specified in seconds GET parameter, or for 1 second if not specified.
func (api *API) getTraceHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sec, err := strconv.ParseFloat(r.FormValue("seconds"), 64)
		if sec <= 0 || err != nil {
			sec = 1
		}

		if durationExceedsWriteTimeout(r, sec) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Go-Pprof", "1")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "profile duration exceeds server's WriteTimeout")
			return nil
		}

		// Set Content Type assuming trace.Start will work,
		// because if it does it starts writing.
		w.Header().Set("Content-Type", "application/octet-stream")
		if err := trace.Start(w); err != nil {
			// trace.Start failed, so no writes yet.
			// Can change header back to text content and send error code.
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Go-Pprof", "1")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Could not enable tracing: %s\n", err)
			return nil
		}
		sleep(w, time.Duration(sec*float64(time.Second)))
		trace.Stop()
		return nil
	}
}

// getCPUProfileHandler responds with the pprof-formatted cpu profile.
// The package initialization registers it as /debug/pprof/profile.
func (api *API) getCPUProfileHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sec, _ := strconv.ParseInt(r.FormValue("seconds"), 10, 64)
		if sec == 0 {
			sec = 30
		}

		if durationExceedsWriteTimeout(r, float64(sec)) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Go-Pprof", "1")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, "profile duration exceeds server's WriteTimeout")
			return nil
		}

		// Set Content Type assuming StartCPUProfile will work,
		// because if it does it starts writing.
		w.Header().Set("Content-Type", "application/octet-stream")
		if err := pprof.StartCPUProfile(w); err != nil {
			// StartCPUProfile failed, so no writes yet.
			// Can change header back to text content
			// and send error code.
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("X-Go-Pprof", "1")
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Could not enable CPU profiling: %s\n", err)
			return nil
		}
		sleep(w, time.Duration(sec)*time.Second)
		pprof.StopCPUProfile()

		return nil
	}
}
