package gorpmapper

import (
	"reflect"
	"sync"
	"text/template"

	"github.com/ovh/symmecrypt"
)

func New() *Mapper {
	m := &Mapper{}
	m.Mapping = make(map[string]TableMapping)
	m.CanonicalFormTemplates.M = make(map[string]*template.Template)
	return m
}

// Mapping is the global var for all registered mapping
type Mapper struct {
	Mapping                map[string]TableMapping
	mappingMutex           sync.Mutex
	CanonicalFormTemplates struct {
		M map[string]*template.Template
		L sync.RWMutex
	}
	encryptionKey symmecrypt.Key
	signatureKey  symmecrypt.Key
	once          sync.Once
}

//Register intialiaze gorp mapping
func (m *Mapper) Register(ms ...TableMapping) {
	m.mappingMutex.Lock()
	defer m.mappingMutex.Unlock()
	for _, t := range ms {
		k := reflect.TypeOf(t.Target).String()
		m.Mapping[k] = t
	}
}

func (m *Mapper) GetTableMapping(i interface{}) (TableMapping, bool) {
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		i = reflect.ValueOf(i).Elem().Interface()
	}
	m.mappingMutex.Lock()
	defer m.mappingMutex.Unlock()
	k := reflect.TypeOf(i).String()
	mapping, has := m.Mapping[k]
	return mapping, has
}

type EncryptedField struct {
	Name   string
	Column string
	Extras []string
}
