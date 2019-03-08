package gorpmapping

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
)

// TableMapping represents a table mapping with gorp
type TableMapping struct {
	Target        interface{}
	Name          string
	AutoIncrement bool
	Keys          []string
}

// New initialize a TableMapping
func New(target interface{}, name string, autoIncrement bool, keys ...string) TableMapping {
	return TableMapping{Target: target, Name: name, AutoIncrement: autoIncrement, Keys: keys}
}

// Mapping is the global var for all registered mapping
var (
	Mapping      = map[string]TableMapping{}
	mappingMutex sync.Mutex
)

//Register intialiaze gorp mapping
func Register(m ...TableMapping) {
	mappingMutex.Lock()
	defer mappingMutex.Unlock()
	for _, t := range m {
		k := fmt.Sprintf("%T", t.Target)
		Mapping[k] = t
	}
}

func getTabbleMapping(i interface{}) (TableMapping, bool) {
	mappingMutex.Lock()
	defer mappingMutex.Unlock()
	k := fmt.Sprintf("%T", i)
	mapping, has := Mapping[k]
	return mapping, has
}

//JSONToNullString returns a valid sql.NullString with json-marshalled i
func JSONToNullString(i interface{}) (sql.NullString, error) {
	if i == nil {
		return sql.NullString{Valid: false}, nil
	}
	b, err := json.Marshal(i)
	if err != nil {
		return sql.NullString{Valid: false}, err
	}
	return sql.NullString{Valid: true, String: string(b)}, nil
}

//JSONNullString sets the holder with unmarshalled sql.NullString
func JSONNullString(s sql.NullString, holder interface{}) error {
	if !s.Valid {
		return nil
	}
	return json.Unmarshal([]byte(s.String), holder)
}
