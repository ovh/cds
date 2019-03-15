package gorpmapping

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	_ "github.com/ovh/symmecrypt/ciphers/hmac"

	"github.com/ovh/cds/sdk"
)

const (
	// ViolateUniqueKeyPGCode is the pg code when duplicating unique key
	ViolateUniqueKeyPGCode = "23505"

	// StringDataRightTruncation is raisedalue is too long for varchar.
	StringDataRightTruncation = "22001"
)

// IDsToQueryString returns a comma separated list of given ids.
func IDsToQueryString(ids []int64) string {
	res := make([]string, len(ids))
	for i := range ids {
		res[i] = fmt.Sprintf("%d", ids[i])
	}
	return strings.Join(res, ",")
}

// Insert value in given db.
func Insert(db gorp.SqlExecutor, i interface{}) error {
	err := db.Insert(i)
	if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ViolateUniqueKeyPGCode:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		case StringDataRightTruncation:
			err = sdk.NewError(sdk.ErrConflict, e)
		}
	}
	return sdk.WithStack(err)
}

// Update value in given db.
func Update(db gorp.SqlExecutor, i interface{}) error {
	_, err := db.Update(i)
	if e, ok := err.(*pq.Error); ok {
		switch e.Code {
		case ViolateUniqueKeyPGCode:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		case StringDataRightTruncation:
			err = sdk.NewError(sdk.ErrInvalidData, e)
		}
	}
	return sdk.WithStack(err)
}

// Delete value in given db.
func Delete(db gorp.SqlExecutor, i interface{}) error {
	_, err := db.Delete(i)
	return sdk.WithStack(err)
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
