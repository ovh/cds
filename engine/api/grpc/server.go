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
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Init initialize all GRPC services
func Init(dbConnectionFactory *database.DBConnectionFactory, port int, tls bool, certFile, keyFile string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	log.Info("Starting GRPC services on port %d", port)

	handlers := &handlers{
		dbConnectionFactory: dbConnectionFactory,
	}

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(handlers.streamInterceptor),
		grpc.UnaryInterceptor(handlers.unaryInterceptor),
	}

	if tls {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)

	RegisterBuildLogServer(grpcServer, handlers)
	RegisterWorkflowQueueServer(grpcServer, handlers)

	return grpcServer.Serve(lis)
}

type key string

const (
	keyWorkerID   key = "worker_id"
	keyWorkerName key = "worker_name"
)

func (h *handlers) streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	c := stream.Context()
	w, err := h.authorize(c)
	if err != nil {
		log.Warning("streamInterceptor> authorize failed : %s", err)
		return err
	}
	m := metadata.Pairs(string(keyWorkerID), w.ID, string(keyWorkerName), w.Name)
	stream.SendHeader(m)

	return handler(srv, stream)
}

func (h *handlers) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	w, err := h.authorize(ctx)
	if err != nil {
		log.Warning("unaryInterceptor> authorize failed : %s", err)
		return nil, err
	}
	ctx = context.WithValue(ctx, keyWorkerID, w.ID)
	ctx = context.WithValue(ctx, keyWorkerName, w.Name)
	return handler(ctx, req)
}

func (h *handlers) authorize(ctx context.Context) (*sdk.Worker, error) {
	if md, ok := metadata.FromContext(ctx); ok {
		if len(md["name"]) > 0 && len(md["token"]) > 0 {
			w, err := auth.GetWorker(h.dbConnectionFactory.GetDBMap(), h.store, md["token"][0])
			if err != nil {
				log.Error("grpc.authorize> Unable to get worker %v:%v => %s", md["name"], md["token"], err)
				return nil, sdk.ErrServiceUnavailable
			}
			if w == nil {
				return nil, sdk.ErrForbidden
			}
			return w, nil
		}
		return nil, sdk.ErrForbidden
	}
	return nil, sdk.ErrForbidden
}
