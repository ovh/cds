package secretbackend

import (
	"net/rpc"
	"path"

	"github.com/hashicorp/go-plugin"
)

// Handshake is the HandshakeConfig used to configure clients and servers.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "CDS_SECRET_BACKEND_MAGIC_COOKIE",
	MagicCookieValue: "Q0RTX1NFQ1JFVF9CQUNLRU5EX01BR0lDX0NPT0tJRQ==",
}

//Serve has to be called in main func of every plugin
func Serve(name string, s Driver) {
	p := CDSSecretBackendPlugin{s}
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			path.Base(name): p,
		},
	})
}

//CDSSecretBackendPlugin is the implementation of plugin.Plugin so we can serve/consume this
type CDSSecretBackendPlugin struct {
	Driver
}

//PluginName is name for the plugin
func (a CDSSecretBackendPlugin) PluginName() string {
	return a.Name()
}

// Server must return an RPC server for this plugin
// type. We construct a CDSActionRPCServer for this.
func (a CDSSecretBackendPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &RPCServer{Impl: a.Driver}, nil
}

// Client must return an implementation of our interface that communicates
// over an RPC client. We return CDSActionRPC for this.
func (a CDSSecretBackendPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	rpcClient := &RPCClient{client: c}
	return rpcClient, nil
}
