package api

// From https://golang.org/src/net/http/pprof/pprof.go
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
)

func (api *API) getDebugProfilesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		profiles := pprof.Profiles()

		profileCounters := map[string]int{}
		for _, p := range profiles {
			if p != nil {
				profileCounters[p.Name()] = p.Count()
			}
		}

		return service.WriteJSON(w, profileCounters, http.StatusOK)
	}
}

func (api *API) getDebugGoroutinesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		grs, err := sdk.ListGoroutines()
		if err != nil {
			return sdk.WithStack(err)
		}

		type goroutineInfo struct {
			State      string
			CreatedBy  string
			SourcePath string
			Caller     string
		}

		var result = make(map[string][]goroutineInfo)

		for _, goroutine := range grs {
			slice := result[goroutine.State]
			slice = append(slice, goroutineInfo{
				State:      goroutine.State,
				CreatedBy:  goroutine.CreatedBy.Func.Raw,
				SourcePath: goroutine.CreatedBy.FullSrcLine(),
				Caller:     goroutine.Stack.Calls[0].Func.Raw,
			})
			result[goroutine.State] = slice
		}

		return service.WriteJSON(w, result, http.StatusOK)
	}
}

// getProfileHandler responds with the pprof-formatted profile named by the request.
func (api *API) getProfileHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := mux.Vars(r)["name"]
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		debug := service.FormInt(r, "debug")
		p := pprof.Lookup(name)
		if p == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "Unknown profile: %s\n", name)
			return nil
		}
		gc := service.FormInt(r, "gc")
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
func (api *API) getTraceHandler() service.Handler {
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
func (api *API) getCPUProfileHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		sec := service.FormInt64(r, "seconds")
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
