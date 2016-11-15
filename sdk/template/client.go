package template

import (
	"log"

	"github.com/hashicorp/go-plugin"
	cdsplugin "github.com/ovh/cds/sdk/plugin"
)

//Client must be used from client side to call the plugin. It's managing plugin instanciation and initializing
type Client struct {
	*plugin.Client
	pluginName   string
	pluginBinary string
	opts         cdsplugin.IOptions
}

//Instance return a fresh instance of the CDSAction plugin dispensed by a RPC server
func (p Client) Instance() (Interface, error) {
	t, err := p.get()
	if err != nil {
		return nil, err
	}
	t.Init(p.opts)
	return t, nil
}

func (p Client) get() (Interface, error) {
	// Connect via RPC
	rpcClient, err := p.Client.Client()
	if err != nil {
		return nil, err
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(p.pluginName)
	if err != nil {
		log.Printf("[CRITICAL] unable to dispense plugin %s (%s) : %s", p.pluginName, p.pluginBinary, err)
		return nil, err
	}

	t := raw.(Interface)
	return t, nil
}
