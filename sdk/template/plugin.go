package template

import (
	"net/rpc"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	cdsplugin "github.com/ovh/cds/sdk/plugin"
)

// Handshake is the HandshakeConfig used to configure clients and servers.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "CDS_TEMPLATE_MAGIC_COOKIE",
	MagicCookieValue: "Q0RTX1RFTVBMQVRFX01BR0lDX0NPT0tJRQ==",
}

//Serve has to be called in main func of every plugin
func Serve(a Interface) {
	p := CDSTemplateExtension{a}
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
			name: CDSTemplateExtension{},
		},
		Cmd: exec.Command(binary),
	})

	options := cdsplugin.Options{
		ID:            id,
		URL:           url,
		TlsSkipVerify: tlsSkipVerify,
	}

	return &Client{client, name, binary, options}
}

//CDSTemplateExtension is the implementation of plugin.Plugin so we can serve/consume this
type CDSTemplateExtension struct {
	Interface
}

//PluginName is name for the plugin
func (a CDSTemplateExtension) PluginName() string {
	return a.Name()
}

// Server must return an RPC server for this plugin
// type. We construct a RPCServer for this.
func (a CDSTemplateExtension) Server(*plugin.MuxBroker) (interface{}, error) {
	return &RPCServer{Impl: a.Interface}, nil
}

// Client must return an implementation of our interface that communicates
// over an RPC client. We return CDSActionRPC for this.
func (a CDSTemplateExtension) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	rpcClient := &RPCClient{client: c}
	return rpcClient, nil
}
