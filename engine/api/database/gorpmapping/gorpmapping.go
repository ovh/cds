package gorpmapping

import (
	"os"
	"sync"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var verbose bool

func init() {
	if os.Getenv("gorp_trace") == "true" {
		verbose = true
	}
}

func debug(format string, args ...interface{}) {
	if verbose {
		log.Debug(format, args...)
	}
}

// TableMapping represents a table mapping with gorp
type TableMapping struct {
	Target        interface{}
	Name          string
	AutoIncrement bool
	Keys          map[string]string
}

// GetKeys returns primary keys columns
func (m TableMapping) GetKeys() []string {
	var keys = make([]string, len(m.Keys))
	var idx int
	for k := range m.Keys {
		keys[idx] = k
		idx++
	}
	return keys
}

// New initialize a TableMapping
func New(t interface{}, n string, b bool, k ...string) TableMapping {
	f := interfaceToValue(t)
	keys := map[string]string{}

	for i := 0; i < f.NumField(); i++ {
		if !f.Field(i).CanInterface() {
			continue
		}
		tag := f.Type().Field(i).Tag.Get("db")
		if sdk.IsInArray(tag, k) {
			keys[tag] = f.Type().Field(i).Name
		}

	}

	return TableMapping{t, n, b, keys}
}

// Mapping is the global var for all registered mapping
var Mapping []TableMapping

//Register intialiaze gorp mapping
func Register(m ...TableMapping) {
	for _, t := range m {
		registerMapping(t)
		registerJSONColumns(t)
	}
}

var mapMappingPerType = struct {
	mutex sync.RWMutex
	data  map[string]TableMapping
}{
	data: map[string]TableMapping{},
}

func registerMapping(m TableMapping) {
	f := interfaceToValue(m.Target)
	mapMappingPerType.mutex.Lock()
	mapMappingPerType.data[f.Type().PkgPath()+"/"+f.Type().Name()] = m
	mapMappingPerType.mutex.Unlock()
	Mapping = append(Mapping, m)
}
