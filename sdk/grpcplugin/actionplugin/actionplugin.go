package actionplugin

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ovh/cds/sdk/grpcplugin"
<<<<<<< HEAD
	"google.golang.org/grpc"
)

type Common struct {
	grpcplugin.Common
	conn     *grpc.ClientConn
	HTTPPort int32
}

=======

	"google.golang.org/grpc"
)

// Common is the common struct of actionplugin
type Common struct {
	grpcplugin.Common
	conn     *grpc.ClientConn //nolint
	HTTPPort int32
}

// Start is useful to start grpcplugin
>>>>>>> feat(api): add grpc plugin action handlers (#3308)
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

<<<<<<< HEAD
=======
// Client gives us a grpcplugin client
>>>>>>> feat(api): add grpc plugin action handlers (#3308)
func Client(ctx context.Context, socket string) (ActionPluginClient, error) {
	conn, err := grpc.DialContext(ctx,
		socket,
		grpc.WithInsecure(),
		grpc.WithDialer(func(address string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", socket, timeout)
<<<<<<< HEAD
		},
		),
=======
		}),
>>>>>>> feat(api): add grpc plugin action handlers (#3308)
	)
	if err != nil {
		return nil, err
	}

	c := NewActionPluginClient(conn)
	return c, nil
}
