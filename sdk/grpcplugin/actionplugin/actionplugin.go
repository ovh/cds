package actionplugin

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	empty "github.com/golang/protobuf/ptypes/empty"
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
	conn, err := grpc.DialContext(ctx,
		socket,
		grpc.WithInsecure(),
		grpc.WithDialer(func(address string, timeout time.Duration) (net.Conn, error) {
			if strings.Contains(socket, ".sock") {
				return net.DialTimeout("unix", socket, timeout)
			}
			return net.DialTimeout("tcp", socket, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	c := NewActionPluginClient(conn)
	return c, nil
}

func (c *Common) WorkerHTTPPort(ctx context.Context, q *WorkerHTTPPortQuery) (*empty.Empty, error) {
	c.HTTPPort = q.Port
	return &empty.Empty{}, nil
}

func Fail(format string, args ...interface{}) (*ActionResult, error) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(msg)
	return &ActionResult{
		Details: msg,
		Status:  "Fail",
	}, nil
}
