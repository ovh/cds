package gorpmapping

import (
	"database/sql"
	"encoding/json"
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
var Mapping []TableMapping

//Register intialiaze gorp mapping
func Register(m ...TableMapping) {
	for _, t := range m {
		Mapping = append(Mapping, t)
	}
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
