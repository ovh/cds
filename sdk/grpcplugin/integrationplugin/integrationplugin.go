package integrationplugin

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
	conn *grpc.ClientConn
}

func Start(ctx context.Context, srv IntegrationPluginServer) error {
	p, ok := srv.(grpcplugin.Plugin)
	if !ok {
		return fmt.Errorf("bad implementation")
	}

	c := p.Instance()
	c.Srv = srv
	c.Desc = &_IntegrationPlugin_serviceDesc
	return p.Start(ctx)
}

func Client(ctx context.Context, socket string) (IntegrationPluginClient, error) {
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

	c := NewIntegrationPluginClient(conn)
	return c, nil
}
