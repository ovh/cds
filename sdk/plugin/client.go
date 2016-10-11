package plugin

import (
	"log"

	"github.com/hashicorp/go-plugin"
)

//Client must be used from client side to call the plugin. It's managing plugin instanciation and initializing
type Client struct {
	*plugin.Client
	pluginName   string
	pluginBinary string
	opts         IOptions
}

//Instance return a fresh instance of the CDSAction plugin dispensed by a RPC server
func (p Client) Instance() (CDSAction, error) {
	action, err := p.get()
	if err != nil {
		return nil, err
	}
	action.Init(p.opts)
	return action, nil
}

func (p Client) get() (CDSAction, error) {
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

	action := raw.(CDSAction)
	return action, nil
}
