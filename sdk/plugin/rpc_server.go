package plugin

//CDSActionRPCServer is the struct called to serve the plugin
type CDSActionRPCServer struct {
	Impl CDSAction
}

//Name serves rpc call to Name()
func (c *CDSActionRPCServer) Name(args interface{}, resp *string) error {
	*resp = c.Impl.Name()
	return nil
}

//Description serves rpc call to Description()
func (c *CDSActionRPCServer) Description(args interface{}, resp *string) error {
	*resp = c.Impl.Description()
	return nil
}

//Author serves rpc call to Author()
func (c *CDSActionRPCServer) Author(args interface{}, resp *string) error {
	*resp = c.Impl.Author()
	return nil
}

//Parameters serves rpc call to Parameters()
func (c *CDSActionRPCServer) Parameters(args interface{}, resp *Parameters) error {
	*resp = c.Impl.Parameters()
	return nil
}

//Run serves rpc call to Run()
func (c *CDSActionRPCServer) Run(args interface{}, resp *Result) error {
	job := args.(IJob)
	*resp = c.Impl.Run(job)
	return nil
}

//Init the rpc plugin
func (c *CDSActionRPCServer) Init(args interface{}, resp *string) error {
	id := args.(IOptions)
	*resp = c.Impl.Init(id)
	return nil
}

//Version of the rpc plugin
func (c *CDSActionRPCServer) Version(args interface{}, resp *string) error {
	*resp = c.Impl.Version()
	return nil
}
