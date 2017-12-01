package defaults

import (
	"crypto/md5"
	"encoding/hex"
	"math/rand"
	"reflect"
	"time"
)

func Factory(variable interface{}) {
	getFactoryFiller().Fill(variable)
}

var factoryFiller *Filler = nil

func getFactoryFiller() *Filler {
	if factoryFiller == nil {
		factoryFiller = newFactoryFiller()
	}

	return factoryFiller
}

func newFactoryFiller() *Filler {
	rand.Seed(time.Now().UTC().UnixNano())

	funcs := make(map[reflect.Kind]FillerFunc, 0)

	funcs[reflect.Bool] = func(field *FieldData) {
		if rand.Intn(1) == 1 {
			field.Value.SetBool(true)
		} else {
			field.Value.SetBool(false)
		}
	}

	funcs[reflect.Int] = func(field *FieldData) {
		field.Value.SetInt(int64(rand.Int()))
	}

	funcs[reflect.Int8] = funcs[reflect.Int]
	funcs[reflect.Int16] = funcs[reflect.Int]
	funcs[reflect.Int32] = funcs[reflect.Int]
	funcs[reflect.Int64] = funcs[reflect.Int]

	funcs[reflect.Float32] = func(field *FieldData) {
		field.Value.SetFloat(rand.Float64())
	}

	funcs[reflect.Float64] = funcs[reflect.Float32]

	funcs[reflect.Uint] = func(field *FieldData) {
		field.Value.SetUint(uint64(rand.Uint32()))
	}

	funcs[reflect.Uint8] = funcs[reflect.Uint]
	funcs[reflect.Uint16] = funcs[reflect.Uint]
	funcs[reflect.Uint32] = funcs[reflect.Uint]
	funcs[reflect.Uint64] = funcs[reflect.Uint]

	funcs[reflect.String] = func(field *FieldData) {
		field.Value.SetString(randomString())
	}

	funcs[reflect.Slice] = func(field *FieldData) {
		if field.Value.Type().Elem().Kind() == reflect.Uint8 {
			if field.Value.Bytes() != nil {
				return
			}

			field.Value.SetBytes([]byte(randomString()))
		}
	}

	funcs[reflect.Struct] = func(field *FieldData) {
		fields := getFactoryFiller().GetFieldsFromValue(field.Value, nil)
		getFactoryFiller().SetDefaultValues(fields)
	}

	return &Filler{FuncByKind: funcs, Tag: "factory"}
}

func randomString() string {
	hash := md5.Sum([]byte(time.Now().UTC().String()))
	return hex.EncodeToString(hash[:])
}
