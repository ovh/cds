package plugin

import (
	"encoding/gob"
)

func init() {
	gob.Register(Result(""))
	gob.Register(ParameterType(""))
	gob.Register(Action{})
	gob.Register(Options{})
	gob.Register(Arguments{})
}

//CDSAction is the standard CDSAction Plugin interface
type CDSAction interface {
	Init(IOptions) string
	Name() string
	Description() string
	Author() string
	Parameters() Parameters
	Run(IAction) Result
}

//IAction is the Run() input args of every plugin
type IAction interface {
	ID() int64
	Arguments() Arguments
}

//Action is the input of the plugin run function
type Action struct {
	IDActionBuild int64
	Args          Arguments
}

func (a Action) ID() int64            { return a.IDActionBuild }
func (a Action) Arguments() Arguments { return a.Args }

//IOptions is
type IOptions interface {
	Hash() string
	GetURL() string
	TLSSkipVerify() bool
}

//Options is
type Options struct {
	ID            string
	URL           string
	TlsSkipVerify bool
}

//Hash prepare authentified requests
func (o Options) Hash() string {
	return o.ID
}

//GetURL prepare authentified requests
func (o Options) GetURL() string {
	return o.URL
}

//TLSSkipVerify returns TLS_SKIP_VERIFY
func (o Options) TLSSkipVerify() bool {
	return o.TlsSkipVerify
}

//Result is the output of the plugin run function
type Result string
type ParameterType string

//Different values for result
const (
	Success                            = "Success"
	Fail                               = "Fail"
	EnvironmentParameter ParameterType = "env"
	PipelineParameter    ParameterType = "pipeline"
	ListParameter        ParameterType = "list"
	NumberParameter      ParameterType = "number"
	PasswordParameter    ParameterType = "password"
	StringParameter      ParameterType = "string"
	TextParameter        ParameterType = "text"
	BooleanParameter     ParameterType = "boolean"
)

type IParameters interface {
	Names() []string
	GetType(string) ParameterType
	GetDescription(string) string
	GetValue(string) string
}

func NewParameters() Parameters {
	p := Parameters{}
	p.Data = map[string]string{}
	p.DataType = map[string]ParameterType{}
	p.DataDescription = map[string]string{}
	return p
}

type Parameters struct {
	Data            map[string]string
	DataType        map[string]ParameterType
	DataDescription map[string]string
}

func (p *Parameters) Add(name string, _type ParameterType, description string, value string) {
	p.Data[name] = value
	p.DataType[name] = _type
	p.DataDescription[name] = description
}

func (p *Parameters) Names() []string {
	keys := []string{}
	for k := range p.Data {
		keys = append(keys, k)
	}
	return keys
}

func (p *Parameters) GetType(k string) ParameterType {
	return p.DataType[k]
}

func (p *Parameters) GetDescription(k string) string {
	return p.DataDescription[k]
}

func (p *Parameters) GetValue(k string) string {
	return p.Data[k]
}

type IArguments interface {
	Get(string) string
	Exists(string) bool
}

type Arguments struct {
	Data map[string]string
}

func (p Arguments) Get(key string) string {
	return p.Data[key]
}

func (p *Arguments) Set(key, value string) {
	p.Data[key] = value
}

func (p Arguments) Exists(key string) bool {
	_, ok := p.Data[key]
	return ok
}
