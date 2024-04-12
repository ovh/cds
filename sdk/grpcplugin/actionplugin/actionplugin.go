package actionplugin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/grpcplugin"
	"github.com/pkg/errors"

	"google.golang.org/grpc"
)

// Common is the common struct of actionplugin
type Common struct {
	grpcplugin.Common
	conn         *grpc.ClientConn //nolint
	HTTPPort     int32
	HTTPClient   cdsclient.HTTPClient
	StreamServer ActionPlugin_StreamServer
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

func (c *Common) NewRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if c.HTTPPort == 0 {
		return nil, errors.Errorf("worker port must not be 0")
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	req, err := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d%s", c.HTTPPort, path), body)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return req.WithContext(ctx), nil
}

func (c *Common) DoRequest(req *http.Request) (*http.Response, error) {
	if c.HTTPClient == nil {
		c.HTTPClient = http.DefaultClient
	}
	return c.HTTPClient.Do(req)
}
