package plugin

import (
	"log"
	"net/rpc"
)

//CDSActionRPC is the struct used by the worker
type CDSActionRPC struct {
	client *rpc.Client
}

//Name makes rpc call to Name()
func (c *CDSActionRPC) Name() string {
	var resp string
	if err := c.client.Call("Plugin.Name", new(interface{}), &resp); err != nil {
		log.Println("[ERROR] Plugin.Name rpc failed")
		panic(err)
	}
	return resp
}

//Description makes rpc call to Description()
func (c *CDSActionRPC) Description() string {
	var resp string
	if err := c.client.Call("Plugin.Description", new(interface{}), &resp); err != nil {
		log.Println("[ERROR] Plugin.Description rpc failed")
		panic(err)
	}
	return resp
}

//Author makes rpc call to Author()
func (c *CDSActionRPC) Author() string {
	var resp string
	if err := c.client.Call("Plugin.Author", new(interface{}), &resp); err != nil {
		log.Println("[ERROR] Plugin.Author rpc failed")
		panic(err)
	}
	return resp
}

//Parameters makes rpc call to Parameters()
func (c *CDSActionRPC) Parameters() Parameters {
	var resp = Parameters{}
	if err := c.client.Call("Plugin.Parameters", new(interface{}), &resp); err != nil {
		log.Println("[ERROR] Plugin.Parameters rpc failed")
		panic(err)
	}
	return resp
}

//Run makes rpc call to Run() on client side
func (c *CDSActionRPC) Run(a IJob) Result {
	var resp Result
	if err := c.client.Call("Plugin.Run", &a, &resp); err != nil {
		log.Println("[ERROR] Plugin.Run rpc failed")
		resp = Fail
	}
	return resp
}

//Init the plugin
func (c *CDSActionRPC) Init(id IOptions) string {
	var resp string
	if err := c.client.Call("Plugin.Init", &id, &resp); err != nil {
		log.Println("[ERROR] Plugin.Init rpc failed")
		resp = Fail
	}
	return resp
}

//Version of the plugin
func (c *CDSActionRPC) Version() string {
	var resp string
	if err := c.client.Call("Plugin.Version", new(interface{}), &resp); err != nil {
		log.Println("[ERROR] Plugin.Version rpc failed")
		panic(err)
	}
	return resp
}
