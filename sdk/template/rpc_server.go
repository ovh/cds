package template

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
)

//RPCServer is the struct called to serve the plugin
type RPCServer struct {
	Impl Interface
}

//Name returns name of the template
func (s *RPCServer) Name(args interface{}, resp *string) error {
	*resp = s.Impl.Name()
	return nil
}

//Description returns the description of the template
func (s *RPCServer) Description(args interface{}, resp *string) error {
	*resp = s.Impl.Description()
	return nil
}

//Author returns the author's name of the template
func (s *RPCServer) Author(args interface{}, resp *string) error {
	*resp = s.Impl.Author()
	return nil
}

//Identifier returns the identifier of the template
func (s *RPCServer) Identifier(args interface{}, resp *string) error {
	*resp = s.Impl.Identifier()
	return nil
}

//Type returns type of the template
func (s *RPCServer) Type(args interface{}, resp *string) error {
	*resp = s.Impl.Type()
	return nil
}

//Parameters returns parameters of the template
func (s *RPCServer) Parameters(args interface{}, resp *[]sdk.TemplateParam) error {
	*resp = s.Impl.Parameters()
	return nil
}

//Apply returns an application instance ready to persist in database
func (s *RPCServer) Apply(args interface{}, resp *sdk.Application) error {
	var err error
	opts := args.(IApplyOptions)
	*resp, err = s.Impl.Apply(opts)
	return err
}

//Init the rpc plugin
func (s *RPCServer) Init(args interface{}, resp *string) error {
	opts := args.(plugin.IOptions)
	*resp = s.Impl.Init(opts)
	return nil
}
