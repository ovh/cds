package secretbackend

//RPCServer is the struct called to serve the plugin
type RPCServer struct {
	Impl Driver
}

//Name serves rpc call to Name()
func (c *RPCServer) Name(args interface{}, resp *string) error {
	*resp = c.Impl.Name()
	return nil
}

//Init the rpc plugin
func (c *RPCServer) Init(args interface{}, resp *string) error {
	opts := args.(MapVar)
	return c.Impl.Init(opts)
}

//GetSecrets serves rpc call to GetSecrets()
func (c *RPCServer) GetSecrets(args interface{}, resp *Secrets) error {
	*resp = c.Impl.GetSecrets()
	return nil
}
