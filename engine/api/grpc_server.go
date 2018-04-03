package api

import (
	"fmt"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/database"
	cdsgrpc "github.com/ovh/cds/engine/api/grpc"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// grpcInit initialize all GRPC services
func grpcInit(dbConnectionFactory *database.DBConnectionFactory, addr string, port int, tls bool, certFile, keyFile string) error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, port))
	if err != nil {
		return err
	}

	log.Info("Starting GRPC services on %s:%d", addr, port)

	grpcHandlers := &grpcHandlers{
		dbConnectionFactory: dbConnectionFactory,
	}

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpcHandlers.streamInterceptor),
		grpc.UnaryInterceptor(grpcHandlers.unaryInterceptor),
	}

	if tls {
		creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = append(opts, grpc.Creds(creds))
	}

	grpcServer := grpc.NewServer(opts...)

	cdsgrpc.RegisterBuildLogServer(grpcServer, grpcHandlers)
	cdsgrpc.RegisterWorkflowQueueServer(grpcServer, grpcHandlers)

	return grpcServer.Serve(lis)
}

type key string

const (
	keyWorkerID    key = "worker_id"
	keyWorkerName  key = "worker_name"
	keyWorkerGroup key = "worker_group"
)

func (h *grpcHandlers) streamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	c := stream.Context()
	w, err := h.authorize(c)
	if err != nil {
		return sdk.WrapError(err, "streamInterceptor> authorize failed ")
	}
	m := metadata.Pairs(string(keyWorkerID), w.ID, string(keyWorkerName), w.Name)
	stream.SendHeader(m)

	return handler(srv, stream)
}

func (h *grpcHandlers) unaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	w, err := h.authorize(ctx)
	if err != nil {
		log.Warning("unaryInterceptor> authorize failed : %s", err)
		return nil, err
	}
	ctx = context.WithValue(ctx, keyWorkerID, w.ID)
	ctx = context.WithValue(ctx, keyWorkerName, w.Name)
	return handler(ctx, req)
}

func (h *grpcHandlers) authorize(ctx context.Context) (*sdk.Worker, error) {
	if md, ok := metadata.FromContext(ctx); ok {
		if len(md["name"]) > 0 && len(md["token"]) > 0 {
			w, err := auth.GetWorker(h.dbConnectionFactory.GetDBMap(), h.store, md["token"][0], md["name"][0])
			if err != nil {
				log.Error("grpc.authorize> Unable to get worker %v:%v => %s", md["name"], md["token"], err)
				return nil, sdk.ErrServiceUnavailable
			}
			if w == nil {
				return nil, sdk.ErrForbidden
			}

			g, _, err := loadPermissionsByGroupID(h.dbConnectionFactory.GetDBMap(), h.store, w.GroupID)
			if err != nil {
				log.Error("grpc.authorize> Unable to get worker group permission: %s", err)
				return nil, sdk.ErrServiceUnavailable
			}
			w.GroupID = g.ID
			return w, nil
		}
		return nil, sdk.ErrForbidden
	}
	return nil, sdk.ErrForbidden
}
