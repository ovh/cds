package actionplugin

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ovh/cds/sdk/grpcplugin"

	"google.golang.org/grpc"
)

// Common is the common struct of actionplugin
type Common struct {
	grpcplugin.Common
	conn     *grpc.ClientConn //nolint
	HTTPPort int32
}

// Start is useful to start grpcplugin
func Start(ctx context.Context, srv ActionPluginServer) error {
	p, ok := srv.(grpcplugin.Plugin)
	if !ok {
		return fmt.Errorf("bad implementation")
	}

	c := p.Instance()
	c.Srv = srv
	c.Desc = &_ActionPlugin_serviceDesc
	return p.Start(ctx)
}

// Client gives us a grpcplugin client
func Client(ctx context.Context, socket string) (ActionPluginClient, error) {
	var conn *grpc.ClientConn
	var err error
	if !strings.Contains(socket, ".socket") {
		conn, err = grpc.Dial(socket, grpc.WithInsecure(), grpc.WithBackoffMaxDelay(500*time.Millisecond))
	} else {
		conn, err = grpc.DialContext(ctx,
			socket,
			grpc.WithInsecure(),
			grpc.WithBackoffMaxDelay(500*time.Millisecond),
			grpc.WithDialer(func(address string, timeout time.Duration) (net.Conn, error) {
				return net.DialTimeout("unix", socket, timeout)
			}),
		)
	}

	if err != nil {
		return nil, err
	}

	c := NewActionPluginClient(conn)
	return c, nil
}
