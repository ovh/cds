package secretbackend

import (
	"os/exec"
	"path"

	"github.com/hashicorp/go-plugin"
	"github.com/ovh/cds/sdk/log"
)

//Client must be used from client side to call the plugin. It's managing plugin instanciation and initializing
type Client struct {
	*plugin.Client
	name         string
	pluginBinary string
	opts         map[string]string
}

//NewClient has to be called every time we nedd to call a plugin
func NewClient(binary string, options map[string]string) *Client {
	name := path.Base(binary)
	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: CDSSecretBackendPlugin{},
		},
		Cmd: exec.Command(binary),
	})

	return &Client{client, name, binary, options}
}

//Instance return a fresh instance of the plugin dispensed by a RPC server
func (p Client) Instance() (Driver, error) {
	backend, err := p.get()
	if err != nil {
		return nil, err
	}
	backend.Init(NewOptions(p.opts))
	return backend, nil
}

func (p Client) get() (Driver, error) {
	// Connect via RPC
	rpcClient, err := p.Client.Client()
	if err != nil {
		return nil, err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(p.name)
	if err != nil {
		log.Error("Unable to dispense plugin %s (%s) : %s", p.name, p.pluginBinary, err)
		return nil, err
	}

	backend := raw.(Driver)
	return backend, nil
}
