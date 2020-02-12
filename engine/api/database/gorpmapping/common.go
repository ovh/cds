package gorpmapping

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/sdk"
)

const (
	// ViolateForeignKeyPGCode is the pg code when violating foreign key
	ViolateForeignKeyPGCode = "23503"

	// ViolateUniqueKeyPGCode is the pg code when duplicating unique key
	ViolateUniqueKeyPGCode = "23505"

	// RowLockedPGCode is the pg code when trying to access to a locked row
	RowLockedPGCode = "55P03"

	// StringDataRightTruncation is raisedalue is too long for varchar.
	StringDataRightTruncation = "22001"
)

// NewQuery returns a new query from given string request.
func NewQuery(q string) Query { return Query{query: q} }

// Query to get gorp entities in database.
type Query struct {
	query     string
	arguments []interface{}
}

// Args store query arguments.
func (q Query) Args(as ...interface{}) Query {
	q.arguments = as
	return q
}

func (q Query) Limit(i int) Query {
	q.query += ` LIMIT ` + strconv.Itoa(i)
	return q
}

func (q Query) String() string {
	return fmt.Sprintf("query: %s - args: %v", q.query, q.arguments)
}

// ToQueryString returns a comma separated list of given ids.
func ToQueryString(target interface{}) string {

	val := reflect.ValueOf(target)
	if reflect.ValueOf(target).Kind() == reflect.Ptr {
		val = val.Elem()
	}

	res := make([]string, val.Len())
	for i := 0; i < val.Len(); i++ {
		res[i] = fmt.Sprintf("%v", val.Index(i).Interface())
	}

	return strings.Join(res, ",")
}

// IDsToQueryString returns a comma separated list of given ids.
func IDsToQueryString(ids []int64) string {
	res := make([]string, len(ids))
	for i := range ids {
		res[i] = fmt.Sprintf("%d", ids[i])
	}
	return strings.Join(res, ",")
}

// IDStringsToQueryString returns a comma separated list of given string ids.
func IDStringsToQueryString(ids []string) string {
	return strings.Join(ids, ",")
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

func dbMappingPKey(i interface{}) (string, string, interface{}, error) {
	mapping, has := getTabbleMapping(i)
	if !has {
		return "", "", nil, sdk.WithStack(fmt.Errorf("unkown entity %T", i))
	}

	if len(mapping.Keys) > 1 {
		return "", "", nil, sdk.WithStack(errors.New("multiple primary key not supported"))
	}

	id := reflectFindValueByTag(i, "db", mapping.Keys[0])

	return mapping.Name, mapping.Keys[0], id, nil
}

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

func LoadTupleByPrimaryKey(db gorp.SqlExecutor, entity string, pk interface{}) (interface{}, error) {
	e, ok := Mapping[entity]
	if !ok {
		return nil, sdk.WithStack(errors.New("unknown entity"))
	}

	newTargetPtr := reflect.New(reflect.TypeOf(e.Target))

	query := NewQuery(fmt.Sprintf(`select * from "%s" where %s::text = $1::text`, e.Name, e.Keys[0])).Args(pk)
	found, err := Get(context.Background(), db, query, newTargetPtr.Interface())
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

		isValid, err := CheckSignature(val.(Canonicaller), s.GetSignature())
		if err != nil {
			return nil, err
		}
		if !isValid {
			return nil, sdk.WithStack(errors.New("corrupted signed entity"))
		}
	}

	return val, nil
}
