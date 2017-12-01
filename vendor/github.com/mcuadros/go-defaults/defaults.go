package defaults

import (
	"reflect"
	"strconv"
	"time"
)

// Applies the default values to the struct object, the struct type must have
// the StructTag with name "default" and the directed value.
//
// Usage
//     type ExampleBasic struct {
//         Foo bool   `default:"true"`
//         Bar string `default:"33"`
//         Qux int8
//     }
//
//      foo := &ExampleBasic{}
//      SetDefaults(foo)
func SetDefaults(variable interface{}) {
	getDefaultFiller().Fill(variable)
}

var defaultFiller *Filler = nil

func getDefaultFiller() *Filler {
	if defaultFiller == nil {
		defaultFiller = newDefaultFiller()
	}

	return defaultFiller
}

func newDefaultFiller() *Filler {
	funcs := make(map[reflect.Kind]FillerFunc, 0)
	funcs[reflect.Bool] = func(field *FieldData) {
		value, _ := strconv.ParseBool(field.TagValue)
		field.Value.SetBool(value)
	}

	funcs[reflect.Int] = func(field *FieldData) {
		value, _ := strconv.ParseInt(field.TagValue, 10, 64)
		field.Value.SetInt(value)
	}

	funcs[reflect.Int8] = funcs[reflect.Int]
	funcs[reflect.Int16] = funcs[reflect.Int]
	funcs[reflect.Int32] = funcs[reflect.Int]
	funcs[reflect.Int64] = funcs[reflect.Int]

	funcs[reflect.Float32] = func(field *FieldData) {
		value, _ := strconv.ParseFloat(field.TagValue, 64)
		field.Value.SetFloat(value)
	}

	funcs[reflect.Float64] = funcs[reflect.Float32]

	funcs[reflect.Uint] = func(field *FieldData) {
		value, _ := strconv.ParseUint(field.TagValue, 10, 64)
		field.Value.SetUint(value)
	}

	funcs[reflect.Uint8] = funcs[reflect.Uint]
	funcs[reflect.Uint16] = funcs[reflect.Uint]
	funcs[reflect.Uint32] = funcs[reflect.Uint]
	funcs[reflect.Uint64] = funcs[reflect.Uint]

	funcs[reflect.String] = func(field *FieldData) {
		field.Value.SetString(field.TagValue)
	}

	funcs[reflect.Slice] = func(field *FieldData) {
		k := field.Value.Type().Elem().Kind()
		switch k {
		case reflect.Uint8:
			if field.Value.Bytes() != nil {
				return
			}
			field.Value.SetBytes([]byte(field.TagValue))
		case reflect.Struct:
			count := field.Value.Len()
			for i := 0; i < count; i++ {
				fields := getDefaultFiller().GetFieldsFromValue(field.Value.Index(i), nil)
				getDefaultFiller().SetDefaultValues(fields)
			}
		}
	}

	funcs[reflect.Struct] = func(field *FieldData) {
		fields := getDefaultFiller().GetFieldsFromValue(field.Value, nil)
		getDefaultFiller().SetDefaultValues(fields)
	}

	types := make(map[TypeHash]FillerFunc, 1)
	types["time.Duration"] = func(field *FieldData) {
		d, _ := time.ParseDuration(field.TagValue)
		field.Value.Set(reflect.ValueOf(d))
	}

	return &Filler{FuncByKind: funcs, FuncByType: types, Tag: "default"}
}
