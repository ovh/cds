package plugin

import (
	"encoding/gob"
)

func init() {
	gob.Register(Result(""))
	gob.Register(ParameterType(""))
	gob.Register(Job{})
	gob.Register(Options{})
	gob.Register(Arguments{})
	gob.Register(Secrets{})
}

//CDSAction is the standard CDSAction Plugin interface
type CDSAction interface {
	Init(IOptions) string
	Version() string
	Name() string
	Description() string
	Author() string
	Parameters() Parameters
	Run(IJob) Result
}

//IJob is the Run() input args of every plugin
type IJob interface {
	ID() int64
	WorkflowNodeRunID() int64
	PipelineBuildID() int64
	StepOrder() int
	WorkerHTTPPort() int
	Arguments() Arguments
	Secrets() Secrets
}

//Job is the input of the plugin run function
type Job struct {
	IDPipelineJobBuild int64
	IDWorkflowNodeRun  int64
	IDPipelineBuild    int64
	Args               Arguments
	Secrts             Secrets
	OrderStep          int
	HTTPPortWorker     int
}

func (j Job) ID() int64                { return j.IDPipelineJobBuild }
func (j Job) Arguments() Arguments     { return j.Args }
func (j Job) Secrets() Secrets         { return j.Secrts }
func (j Job) PipelineBuildID() int64   { return j.IDPipelineBuild }
func (j Job) WorkflowNodeRunID() int64 { return j.IDWorkflowNodeRun }
func (j Job) StepOrder() int           { return j.OrderStep }
func (j Job) WorkerHTTPPort() int      { return j.HTTPPortWorker }

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
	keys := make([]string, len(p.Data))
	i := 0
	for k := range p.Data {
		keys[i] = k
		i++
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

// Arguments type, key: var name, value: value of argument
type Arguments struct {
	Data map[string]string
}

// Get returns an argument from key
func (p Arguments) Get(key string) string {
	return p.Data[key]
}

// Set sets an argument with key / value
func (p *Arguments) Set(key, value string) {
	p.Data[key] = value
}

// Exists returns true if argument exists
func (p Arguments) Exists(key string) bool {
	_, ok := p.Data[key]
	return ok
}

// Secrets type, key: var name, value: value of secret
type Secrets struct {
	Data map[string]string
}

// Get returns a secret from key
func (p Secrets) Get(key string) string {
	return p.Data[key]
}

// Set sets a secret with key / value
func (p *Secrets) Set(key, value string) {
	p.Data[key] = value
}

// Exists returns true if secret exists
func (p Secrets) Exists(key string) bool {
	_, ok := p.Data[key]
	return ok
}
