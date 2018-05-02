// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pprofutil

import (
	"runtime/pprof"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
)

// UnaryServerInterceptor allows you to profile gRPC server handlers.
//
// Prrof data can be filtered by "grpc.method" tags once instrumented by this interceptor.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer pprof.SetGoroutineLabels(ctx)
		ctx = pprof.WithLabels(ctx, pprof.Labels("grpc.method", info.FullMethod))
		pprof.SetGoroutineLabels(ctx)
		return handler(ctx, req)
	}
}
