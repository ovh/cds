package storageplugin

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ovh/cds/sdk/grpcplugin"
	"google.golang.org/grpc"
)

type Common struct {
	grpcplugin.Common
}

func Start(ctx context.Context, srv StoragePluginServer) error {
	p, ok := srv.(grpcplugin.Plugin)
	if !ok {
		return fmt.Errorf("bad implementation")
	}

	c := p.Instance()
	c.Srv = srv
	c.Desc = &_StoragePlugin_serviceDesc
	return p.Start(ctx)
}

func Client(ctx context.Context, socket string) (StoragePluginClient, error) {
	conn, err := grpc.DialContext(ctx,
		socket,
		grpc.WithInsecure(),
		grpc.WithDialer(func(address string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", socket, timeout)
		},
		),
	)
	if err != nil {
		return nil, err
	}

	c := NewStoragePluginClient(conn)
	return c, nil
}
