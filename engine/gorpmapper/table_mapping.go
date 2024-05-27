package gorpmapper

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/rockbears/yaml"

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
	EncryptedFields []EncryptedField
}

func deepFields(iface interface{}) []reflect.StructField {
	fields := make([]reflect.StructField, 0)
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)

	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		dbTag, hasDBTag := ift.Field(i).Tag.Lookup("db")
		tagValues := strings.Split(dbTag, ",")
		if len(tagValues) > 0 && tagValues[0] == "-" {
			continue
		}

		switch {
		case v.Kind() == reflect.Struct && !hasDBTag:
			fields = append(fields, deepFields(v.Interface())...)
		default:
			fields = append(fields, ift.Field(i))
		}
	}

	return fields
}

// NewTableMapping initialize a TableMapping.
func (m *Mapper) NewTableMapping(target interface{}, name string, autoIncrement bool, keys ...string) TableMapping {
	v := sdk.ValueFromInterface(target)

	if v.Kind() != reflect.Struct {
		err := fmt.Errorf("TableMapping error: target (%T) must be a struct", target)
		panic(err)
	}

	var (
		encryptedEntity bool
		encryptedFields []EncryptedField
		signedEntity    bool
	)

	fields := deepFields(target)
	for i := 0; i < len(fields); i++ {
		dbTag, okDBTag := fields[i].Tag.Lookup("db")
		if !okDBTag {
			continue
		}

		tagValues := strings.Split(dbTag, ",")
		if len(tagValues) == 0 {
			continue
		}

		gmTag, okGMTag := fields[i].Tag.Lookup("gorpmapping")
		if okGMTag {
			tagValues := strings.Split(gmTag, ",")
			if len(tagValues) == 0 {
				continue
			}
			column := strings.SplitN(dbTag, ",", 2)[0]

			if tagValues[0] == "encrypted" {
				encryptedEntity = true
				encryptedFields = append(encryptedFields, EncryptedField{
					Name:   fields[i].Name,
					Extras: tagValues[1:],
					Column: column,
				})
			}
		}
	}

	for i := 0; i < v.NumField(); i++ {
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
			_, _ = h.Write(f.Bytes())
			bs := h.Sum(nil)
			sha := fmt.Sprintf("%x", bs)

			t := template.New(sha)
			var err error

			t = t.Funcs(template.FuncMap{
				"printf": func(i interface{}) string {
					return fmt.Sprintf("%v", i)
				},
				"print": func(i interface{}) string {
					return fmt.Sprintf("%v", err)
				},
				"printDate": func(i time.Time) string {
					return i.In(time.UTC).Format(time.RFC3339)
				},
				"md5sum": func(i interface{}) string {
					var dataBts []byte
					dataString, is := i.(string)
					if !is {
						dataBts, _ = yaml.Marshal(i)
					} else {
						dataBts = []byte(dataString)
					}
					return fmt.Sprintf("%x", md5.Sum(dataBts))
				},
				"hash": func(i interface{}) string {
					var dataBts []byte
					dataString, is := i.(string)
					if !is {
						dataBts, _ := yaml.Marshal(i)
						dataString = string(dataBts)
					} else {
						dataBts = []byte(dataString)
					}
					return fmt.Sprintf("%x", md5.Sum(dataBts))
				},
			})

			t, err = t.Parse(f.String())
			if err != nil {
				err := fmt.Errorf("TableMapping error: target (%T) canonical function \"%s\" is invalid: %v", target, f.String(), err)
				panic(err)
			}

			// Test the template
			var out = bytes.Buffer{}
			if err := t.Execute(&out, target); err != nil {
				err := fmt.Errorf("TableMapping error: target (%T) template error: %v", target, err)
				panic(err)
			}

			m.CanonicalFormTemplates.L.Lock()
			m.CanonicalFormTemplates.M[sha] = t
			m.CanonicalFormTemplates.L.Unlock()
		}
	}

	return TableMapping{
		Target:          target,
		Name:            name,
		AutoIncrement:   autoIncrement,
		Keys:            keys,
		SignedEntity:    signedEntity,
		EncryptedEntity: encryptedEntity,
		EncryptedFields: encryptedFields,
	}
}
