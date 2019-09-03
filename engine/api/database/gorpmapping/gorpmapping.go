package gorpmapping

import (
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/ovh/cds/sdk"
)

// TableMapping represents a table mapping with gorp
type TableMapping struct {
	Target          interface{}
	Name            string
	AutoIncrement   bool
	SignedEntity    bool
	Keys            []string
	EncryptedEntity bool
	EncryptedFields []string
}

// New initialize a TableMapping.
func New(target interface{}, name string, autoIncrement bool, keys ...string) TableMapping {
	v := sdk.ValueFromInterface(target)

	if v.Kind() != reflect.Struct {
		err := fmt.Errorf("TableMapping error: target (%T) must be a struct", target)
		panic(err)
	}

	var (
		encryptedEntity bool
		encryptedFields []string
	)
	var signedEntity bool
	for i := 0; i < v.NumField(); i++ {
		dbTag, ok := v.Type().Field(i).Tag.Lookup("gorpmapping")
		if ok {
			if sdk.IsInArray("encrypted", strings.Split(dbTag, ",")) {
				encryptedEntity = true
				encryptedFields = append(encryptedFields, v.Type().Field(i).Name)
			}
		}

		if v.Type().Field(i).Name == reflect.TypeOf(SignedEntity{}).Name() {
			signedEntity = true
		}
	}

	if signedEntity {
		x, ok := target.(Canonicaller)
		if !ok {
			err := fmt.Errorf("TableMapping error: target (%T) must implement Canonicaller interface because it's a signed entity", target)
			panic(err)
		}

		if _, ok := target.(Signed); !ok {
			err := fmt.Errorf("TableMapping error: target (%T) must implement Signed interface because it's a signed entity", target)
			panic(err)
		}

		tmplStrFuncs := x.Canonical()
		for _, f := range tmplStrFuncs {
			h := sha1.New()
			h.Write(f.Bytes())
			bs := h.Sum(nil)
			sha := fmt.Sprintf("%x", bs)

			t := template.New(sha)
			var err error

			t = t.Funcs(template.FuncMap{
				"print": func(i interface{}) string {
					return fmt.Sprintf("%v", err)
				},
				"printDate": func(i time.Time) string {
					return i.In(time.UTC).Format(time.RFC3339)
				},
			})

			t, err = t.Parse(f.String())
			if err != nil {
				err := fmt.Errorf("TableMapping error: target (%T) canonical function \"%s\" is invalid: %v", target, f.String(), err)
				panic(err)
			}
			CanonicalFormTemplates.l.Lock()
			CanonicalFormTemplates.m[sha] = t
			CanonicalFormTemplates.l.Unlock()
		}

	}

	var m = TableMapping{
		Target:          target,
		Name:            name,
		AutoIncrement:   autoIncrement,
		Keys:            keys,
		SignedEntity:    signedEntity,
		EncryptedEntity: encryptedEntity,
		EncryptedFields: encryptedFields,
	}

	return m
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
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		i = reflect.ValueOf(i).Elem().Interface()
	}
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
