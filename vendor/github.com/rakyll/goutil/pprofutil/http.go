// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package pprofutil contains helpers for runtime/pprof.
*/
package pprofutil

import (
	"context"
	"net/http"
	"runtime/pprof"
)

// LabelHandler adds "http-path" profiler label to the given handler.
// If you want to start new goroutines from h, propagate the labels by
// passing handler's incoming request's context.
func LabelHandler(h http.Handler) http.Handler {
	return &labelHandler{orig: h}
}

// LabelHandlerFunc adds "http-path" profiler label to the given handler function.
// If you want to start new goroutines from h, propagate the labels by passing r.Context().
func LabelHandlerFunc(fn func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return &labelHandler{orig: http.HandlerFunc(fn)}
}

type labelHandler struct {
	orig http.Handler
}

func (l *labelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	labels := pprof.Labels("http-path", r.URL.Path)
	pprof.Do(r.Context(), labels, func(ctx context.Context) {
		l.orig.ServeHTTP(w, r)
	})
}
