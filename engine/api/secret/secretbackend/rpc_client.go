package secretbackend

import (
	"net/rpc"

	"github.com/ovh/cds/sdk/log"
)

//RPCClient is the struct used by the CDS engine
type RPCClient struct {
	client *rpc.Client
}

//Init the plugin
func (c *RPCClient) Init(opts MapVar) error {
	var resp string
	err := c.client.Call("Plugin.Init", &opts, &resp)
	if err != nil {
		log.Critical("[ERROR] SecretBackend.Init rpc failed: %s", err)
	}
	return err
}

//Name makes rpc call to Name()
func (c *RPCClient) Name() string {
	var resp string
	err := c.client.Call("Plugin.Name", new(interface{}), &resp)
	if err != nil {
		log.Critical("[ERROR] SecretBackend.Name rpc failed: %s", err)
		panic(err)
	}
	return resp
}

//GetSecrets makes rpc call to GetSecrets()
func (c *RPCClient) GetSecrets() Secrets {
	var resp = NewSecrets(map[string]string{})
	err := c.client.Call("Plugin.GetSecrets", new(interface{}), &resp)
	if err != nil {
		resp.Error = err
		log.Critical("[ERROR] SecretBackend.GetSecrets rpc failed: %s", err)
	}
	return *resp
}
