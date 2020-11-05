package service

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"runtime/pprof"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"
)

func GetAllProfilesHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		profiles := pprof.Profiles()
		profileCounters := map[string]int{}
		for _, p := range profiles {
			if p != nil {
				profileCounters[p.Name()] = p.Count()
			}
		}
		return WriteJSON(w, profileCounters, http.StatusOK)
	}
}

func GetProfileHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := mux.Vars(r)["name"]
		if name == "" {
			return sdk.ErrWrongRequest
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		debug := FormInt(r, "debug")
		p := pprof.Lookup(name)
		if p == nil {
			w.WriteHeader(404)
			fmt.Fprintf(w, "Unknown profile: %s\n", name)
			return nil
		}
		gc := FormInt(r, "gc")
		if name == "heap" && gc > 0 {
			runtime.GC()
		}
		p.WriteTo(w, debug)
		return nil
	}
}
