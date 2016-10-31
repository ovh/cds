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
	Apply(opts IApplyOptions) (sdk.Application, error)
}

type MapVar interface {
	All() map[string]string
	Get(string) string
}

type Parameters struct {
	Data map[string]string
}

type IApplyOptions interface {
	ProjetKey() string
	ApplicationName() string
	Parameters() MapVar
}

type ApplyOptions struct {
	ProjKey string
	AppName string
	Params  MapVar
}

func NewApplyOptions(proj, app string, params Parameters) ApplyOptions {
	return ApplyOptions{
		ProjKey: proj,
		AppName: app,
		Params:  params,
	}
}

func (o ApplyOptions) ProjetKey() string {
	return o.ProjKey
}

func (o ApplyOptions) ApplicationName() string {
	return o.AppName
}

func (o ApplyOptions) Parameters() MapVar {
	return o.Params
}

func NewParameters(d map[string]string) *Parameters {
	return &Parameters{Data: d}
}

func (d Parameters) All() map[string]string {
	return d.Data
}

func (d Parameters) Get(k string) string {
	return d.Data[k]
}
