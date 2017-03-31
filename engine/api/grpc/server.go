package grpc

import (
	"fmt"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/database"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// Init initialize all GRPC services
func Init(port int, tls bool, certFile, keyFile string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	log.Notice("Starting GRPC services on port %d", port)

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(streamInterceptor),
		grpc.UnaryInterceptor(unaryInterceptor),
	}

	if tls {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)

	handlers := &handlers{}
	RegisterBuildLogServer(grpcServer, handlers)

	return grpcServer.Serve(lis)
}

func streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if err := authorize(stream.Context()); err != nil {
		log.Warning("streamInterceptor> authorize failed : %s", err)
		return err
	}
	return handler(srv, stream)
}

func unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	if err := authorize(ctx); err != nil {
		log.Warning("unaryInterceptor> authorize failed : %s", err)
		return nil, err
	}
	return handler(ctx, req)
}

func authorize(ctx context.Context) error {
	if md, ok := metadata.FromContext(ctx); ok {
		if len(md["name"]) > 0 && len(md["token"]) > 0 {
			w, err := auth.GetWorker(database.GetDBMap(), md["token"][0])
			if err != nil {
				log.Critical("grpc.authorize> Unable to get worker %v:%v => %s", md["name"], md["token"], err)
				return sdk.ErrServiceUnavailable
			}
			if w == nil {
				return sdk.ErrForbidden
			}
			return nil
		}
		return sdk.ErrForbidden
	}
	return sdk.ErrForbidden
}
