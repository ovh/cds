package template

import (
	"encoding/gob"

	"github.com/ovh/cds/sdk/plugin"

	"github.com/ovh/cds/sdk"
)

func init() {
	gob.Register(ApplyOptions{})
	gob.Register(Parameters{})
	gob.Register(sdk.Application{})
	gob.Register(sdk.TemplateParam{})
}

//Interface is the interface for template extensions
type Interface interface {
	Init(plugin.IOptions) string
	Name() string
	Description() string
	Identifier() string
	Author() string
	Type() string
	Parameters() []sdk.TemplateParam
	ActionsNeeded() []string
	Apply(opts IApplyOptions) (sdk.Application, error)
}

//MapVar is an interface for map[string]string
type MapVar interface {
	All() map[string]string
	Get(string) string
}

//Parameters represents the parameters map expected bu the template
type Parameters struct {
	Data map[string]string
}

//IApplyOptions represents interface for Apply func arg
type IApplyOptions interface {
	ProjetKey() string
	ApplicationName() string
	Parameters() MapVar
}

//ApplyOptions represents struct for Apply func arg
type ApplyOptions struct {
	ProjKey string
	AppName string
	Params  MapVar
}

//NewApplyOptions instanciate a ApplyOptions struct
func NewApplyOptions(proj, app string, params Parameters) ApplyOptions {
	return ApplyOptions{
		ProjKey: proj,
		AppName: app,
		Params:  params,
	}
}

//ProjetKey returns the project key
func (o ApplyOptions) ProjetKey() string {
	return o.ProjKey
}

//ApplicationName returns the application name
func (o ApplyOptions) ApplicationName() string {
	return o.AppName
}

//Parameters returns the list of parameters
func (o ApplyOptions) Parameters() MapVar {
	return o.Params
}

//NewParameters instanciates a parameters struct
func NewParameters(d map[string]string) *Parameters {
	return &Parameters{Data: d}
}

//All returns the map
func (d Parameters) All() map[string]string {
	return d.Data
}

//Get returns the value in the map for the key k
func (d Parameters) Get(k string) string {
	return d.Data[k]
}
