package gorpmapper

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/sdk"
)

type IDs pq.Int64Array

// And returns a new AND expression from given ones.
func And(es ...string) string {
	if len(es) == 0 {
		return "true"
	}
	return "(" + strings.Join(es, " AND ") + ")"
}

// ArgsMap represents the map of named sql args.
type ArgsMap map[string]interface{}

// Merge returns a merged map from current and another.
func (a ArgsMap) Merge(other ArgsMap) ArgsMap {
	if a == nil {
		a = make(ArgsMap)
	}
	for k, v := range other {
		a[k] = v
	}
	return a
}

func reflectFindValueByTag(i interface{}, tagKey, tagValue string) interface{} {
	val := reflect.ValueOf(i)
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		val = val.Elem()
	}
	var res interface{}
	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		if typeField.Anonymous {
			res = reflectFindValueByTag(valueField.Interface(), tagKey, tagValue)
			if res != nil {
				return res
			}
		}
		tag := typeField.Tag
		column := tag.Get(tagKey)
		column = strings.SplitN(column, ",", 2)[0]
		if column == tagValue {
			return valueField.Interface()
		}
	}
	return res
}

func reflectFindFieldTagValue(i interface{}, field, tagKey string) string {
	val := reflect.ValueOf(i)
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		val = val.Elem()
	}
	valueField := val.FieldByName(field)
	typeField, _ := val.Type().FieldByName(field)
	if typeField.Anonymous {
		res := reflectFindFieldTagValue(valueField.Interface(), field, tagKey)
		if res != "" {
			return res
		}
	}
	tag := typeField.Tag
	column := tag.Get(tagKey)
	return column
}

func (m *Mapper) loadTupleByPrimaryKey(db gorp.SqlExecutor, entity string, pk interface{}, lock bool, opts ...GetOptionFunc) (interface{}, error) {
	e, ok := m.Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(fmt.Errorf("unknown entity %s", entity))
	}

	newTargetPtr := reflect.New(reflect.TypeOf(e.Target))

	var query = NewQuery(fmt.Sprintf(`SELECT * FROM "%s" WHERE %s::text = $1::text`, e.Name, e.Keys[0])).Args(pk)
	if lock {
		query = NewQuery(fmt.Sprintf(`SELECT * FROM "%s" WHERE %s::text = $1::text FOR UPDATE NO WAIT`, e.Name, e.Keys[0])).Args(pk)
	}
	found, err := m.Get(context.Background(), db, query, newTargetPtr.Interface(), opts...)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, sdk.WithStack(sdk.ErrNotFound)
	}

	val := newTargetPtr.Interface()

	if e.SignedEntity {
		s, ok := val.(Signed)
		if !ok {
			return nil, sdk.WithStack(errors.New("invalid signed entity"))
		}

		isValid, err := m.CheckSignature(val.(Canonicaller), s.GetSignature())
		if err != nil {
			return nil, err
		}
		if !isValid {
			return nil, sdk.WithStack(errors.New("corrupted signed entity"))
		}
	}

	return val, nil
}

func (m *Mapper) LoadTupleByPrimaryKey(db gorp.SqlExecutor, entity string, pk interface{}, opts ...GetOptionFunc) (interface{}, error) {
	return m.loadTupleByPrimaryKey(db, entity, pk, false, opts...)
}

func (m *Mapper) LoadAndLockTupleByPrimaryKey(db gorp.SqlExecutor, entity string, pk interface{}, opts ...GetOptionFunc) (interface{}, error) {
	return m.loadTupleByPrimaryKey(db, entity, pk, true, opts...)
}
