package template

import (
	"log"
	"net/rpc"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

//RPCClient is the client for the template extension
type RPCClient struct {
	client *rpc.Client
}

//Init initializes the template extension
func (c *RPCClient) Init(opts plugin.IOptions) string {
	var resp string
	err := c.client.Call("Plugin.Init", &opts, &resp)
	if err != nil {
		log.Println("[ERROR] Plugin.Init rpc failed")
		panic(err)
	}
	return resp
}

//Name returns the name of the template extension
func (c *RPCClient) Name() string {
	var resp string
	err := c.client.Call("Plugin.Name", new(interface{}), &resp)
	if err != nil {
		log.Println("[ERROR] Plugin.Name rpc failed")
		panic(err)
	}
	return resp
}

//Description returns the description of the template extension
func (c *RPCClient) Description() string {
	var resp string
	err := c.client.Call("Plugin.Description", new(interface{}), &resp)
	if err != nil {
		log.Println("[ERROR] Plugin.Description rpc failed")
		panic(err)
	}
	return resp
}

//Identifier returns the identifier of the template extension
func (c *RPCClient) Identifier() string {
	var resp string
	err := c.client.Call("Plugin.Identifier", new(interface{}), &resp)
	if err != nil {
		log.Println("[ERROR] Plugin.Identifier rpc failed")
		panic(err)
	}
	return resp
}

//Author returns the author's name the the template extension
func (c *RPCClient) Author() string {
	var resp string
	err := c.client.Call("Plugin.Author", new(interface{}), &resp)
	if err != nil {
		log.Println("[ERROR] Plugin.Author rpc failed")
		panic(err)
	}
	return resp
}

//Type returns the type of the template type
func (c *RPCClient) Type() string {
	var resp string
	err := c.client.Call("Plugin.Type", new(interface{}), &resp)
	if err != nil {
		log.Println("[ERROR] Plugin.Type rpc failed")
		panic(err)
	}
	return resp
}

//Parameters returns the list of template parameters
func (c *RPCClient) Parameters() []sdk.TemplateParam {
	var resp []sdk.TemplateParam
	err := c.client.Call("Plugin.Parameters", new(interface{}), &resp)
	if err != nil {
		log.Println("[ERROR] Plugin.Name rpc failed")
		panic(err)
	}
	return resp
}

//Apply create a fresh new CDS application from the template extension
func (c *RPCClient) Apply(opts IApplyOptions) (sdk.Application, error) {
	var resp sdk.Application
	err := c.client.Call("Plugin.Apply", &opts, &resp)
	if err != nil {
		log.Println("[ERROR] Plugin.Apply rpc failed")
	}
	return resp, err
}
