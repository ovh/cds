package sdk

import (
	"reflect"
	"runtime"
	"strings"
)

func GetFuncName(i interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	name = strings.Replace(name, ".func1", "", 1)
	name = strings.Replace(name, ".1", "", 1)
	name = strings.Replace(name, "github.com/ovh/cds/engine/", "", 1)
	return name
}

// From https://github.com/fsamin/go-dump/blob/master/helper.go
// Apache-2.0 License
// https://github.com/fsamin/go-dump/blob/master/LICENSE.md

func ValueFromInterface(i interface{}) reflect.Value {
	var f reflect.Value
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		f = reflect.ValueOf(i).Elem()
	} else {
		f = reflect.ValueOf(i)
	}

	if f.Kind() == reflect.Interface {
		if reflect.ValueOf(f.Interface()).Kind() == reflect.Ptr {
			f = reflect.ValueOf(f.Interface()).Elem()
		} else {
			f = reflect.ValueOf(f.Interface())
		}
	}
	return f
}

func ValidAndNotEmpty(v reflect.Value) bool {
	if v.IsValid() && v.CanInterface() {
		if v.Kind() == reflect.String {
			return v.String() != ""
		}
		return true
	}
	return false
}

// ReflectFieldByTag returns a pointer to a value corresponding to a tag
// For instance:
//
// 		sdk.ReflectFieldByTag(&Configuration{}, "toml", "api.database.name")
//
// 		Search for a field tagged 'toml:api' that is a struct.
//		is this value, there is a field tagged 'toml:database' that is a struct.
//      finally in this value, there a field tagged 'toml:name'. We return a pointer to this field.
func ReflectFieldByTag(i interface{}, tagKey, tagValues string) interface{} {
	if reflect.ValueOf(i).Kind() != reflect.Ptr {
		panic("is not a pointer")
	}

	splittedTagValues := strings.Split(tagValues, ".")
	tagValue := splittedTagValues[0]

	valPtr := reflect.ValueOf(i)
	valElem := valPtr.Elem()

	if valElem.Kind() == reflect.Ptr {
		return ReflectFieldByTag(valElem.Interface(), tagKey, tagValues)
	}

	if valElem.Kind() != reflect.Struct {
		return nil
	}

	for idx := 0; idx < valElem.Type().NumField(); idx++ {
		field := valElem.Type().Field(idx)
		valField := valElem.Field(idx)

		t, ok := field.Tag.Lookup(tagKey)
		if ok && t == tagValue && len(splittedTagValues) == 1 {
			if valField.Kind() == reflect.Ptr {
				return valField.Interface()
			}
			return valField.Addr().Interface()

		} else if len(splittedTagValues) > 1 {
			r := ReflectFieldByTag(valField.Addr().Interface(), tagKey, strings.Join(splittedTagValues[1:], "."))
			if r != nil {
				return r
			}
		}
	}
	return nil
}
