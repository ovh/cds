package secretbackend

import "encoding/gob"

func init() {
	gob.Register(Options{})
	gob.Register(Secrets{})
	gob.Register(SecretError(""))
}

//Driver is a plugin interface for retrieve CDS secrets
type Driver interface {
	Init(MapVar) error
	Name() string
	GetSecrets() Secrets
}

type MapVar interface {
	All() map[string]string
	Get(string) string
}

type Options struct {
	Data map[string]string
}

func NewOptions(d map[string]string) *Options {
	return &Options{Data: d}
}

func (d Options) All() map[string]string {
	return d.Data
}

func (d Options) Get(k string) string {
	return d.Data[k]
}

func NewSecrets(d map[string]string) *Secrets {
	return &Secrets{Data: d}
}

type Secrets struct {
	Data  map[string]string
	Error error
}

type SecretError string

func (e SecretError) Error() string {
	return string(e)
}

func Error(err error) *SecretError {
	e := SecretError(err.Error())
	return &e
}

func (d Secrets) All() (map[string]string, error) {
	if d.Error != nil {
		return nil, d.Error
	}
	return d.Data, nil
}

func (d Secrets) Err() error {
	return d.Error
}

func (d Secrets) Get(k string) (string, error) {
	if d.Error != nil {
		return "", d.Error
	}
	return d.Data[k], nil
}
