package plugin

import (
	"net/rpc"
	"os/exec"

	"github.com/hashicorp/go-plugin"
)

// Handshake is the HandshakeConfig used to configure clients and servers.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "CDS_PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "Q0RTX1BMVUdJTl9NQUdJQ19DT09LSUU=",
}

//Serve has to be called in main func of every plugin
func Serve(a CDSAction) {
	p := CDSActionPlugin{a}
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			p.PluginName(): p,
		},
	})

}

//NewClient has to be called every time we nedd to call a plugin
func NewClient(name, binary, id, url string, tlsSkipVerify bool) *Client {
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: CDSActionPlugin{},
		},
		Cmd: exec.Command(binary),
	})

	options := Options{
		ID:            id,
		URL:           url,
		TlsSkipVerify: tlsSkipVerify,
	}

	return &Client{client, name, binary, options}
}

//CDSActionPlugin is the implementation of plugin.Plugin so we can serve/consume this
type CDSActionPlugin struct {
	CDSAction
}

//PluginName is name for the plugin
func (a CDSActionPlugin) PluginName() string {
	return a.Name()
}

// Server must return an RPC server for this plugin
// type. We construct a CDSActionRPCServer for this.
func (a CDSActionPlugin) Server(*plugin.MuxBroker) (interface{}, error) {
	return &CDSActionRPCServer{Impl: a.CDSAction}, nil
}

// Client must return an implementation of our interface that communicates
// over an RPC client. We return CDSActionRPC for this.
func (a CDSActionPlugin) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	rpcClient := &CDSActionRPC{client: c}
	return rpcClient, nil
}
