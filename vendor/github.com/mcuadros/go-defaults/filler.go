package defaults

import (
	"fmt"
	"reflect"
)

type FieldData struct {
	Field    reflect.StructField
	Value    reflect.Value
	TagValue string
	Parent   *FieldData
}

type FillerFunc func(field *FieldData)

// Filler contains all the functions to fill any struct field with any type
// allowing to define function by Kind, Type of field name
type Filler struct {
	FuncByName map[string]FillerFunc
	FuncByType map[TypeHash]FillerFunc
	FuncByKind map[reflect.Kind]FillerFunc
	Tag        string
}

// Fill apply all the functions contained on Filler, setting all the possible
// values
func (f *Filler) Fill(variable interface{}) {
	fields := f.getFields(variable)
	f.SetDefaultValues(fields)
}

func (f *Filler) getFields(variable interface{}) []*FieldData {
	valueObject := reflect.ValueOf(variable).Elem()

	return f.GetFieldsFromValue(valueObject, nil)
}

func (f *Filler) GetFieldsFromValue(valueObject reflect.Value, parent *FieldData) []*FieldData {
	typeObject := valueObject.Type()

	count := valueObject.NumField()
	var results []*FieldData
	for i := 0; i < count; i++ {
		value := valueObject.Field(i)
		field := typeObject.Field(i)

		if value.CanSet() {
			results = append(results, &FieldData{
				Value:    value,
				Field:    field,
				TagValue: field.Tag.Get(f.Tag),
				Parent:   parent,
			})
		}
	}

	return results
}

func (f *Filler) SetDefaultValues(fields []*FieldData) {
	for _, field := range fields {
		if f.isEmpty(field) {
			f.SetDefaultValue(field)
		}
	}
}

func (f *Filler) isEmpty(field *FieldData) bool {
	switch field.Value.Kind() {
	case reflect.Bool:
		return !field.Value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Value.Int() == 0
	case reflect.Float32, reflect.Float64:
		return field.Value.Float() == .0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Value.Uint() == 0
	case reflect.Slice:
		switch field.Value.Type().Elem().Kind() {
		case reflect.Struct:
			// always assume the structs in the slice is empty and can be filled
			// the actually struct filling logic should take care of the rest
			return true
		default:
			return field.Value.Len() == 0
		}
	case reflect.String:
		return field.Value.String() == ""
	}
	return true
}

func (f *Filler) SetDefaultValue(field *FieldData) {
	getters := []func(field *FieldData) FillerFunc{
		f.getFunctionByName,
		f.getFunctionByType,
		f.getFunctionByKind,
	}

	for _, getter := range getters {
		filler := getter(field)
		if filler != nil {
			filler(field)
			return
		}
	}

	return
}

func (f *Filler) getFunctionByName(field *FieldData) FillerFunc {
	if f, ok := f.FuncByName[field.Field.Name]; ok == true {
		return f
	}

	return nil
}

func (f *Filler) getFunctionByType(field *FieldData) FillerFunc {
	if f, ok := f.FuncByType[GetTypeHash(field.Field.Type)]; ok == true {
		return f
	}

	return nil
}

func (f *Filler) getFunctionByKind(field *FieldData) FillerFunc {
	if f, ok := f.FuncByKind[field.Field.Type.Kind()]; ok == true {
		return f
	}

	return nil
}

// TypeHash is a string representing a reflect.Type following the next pattern:
// <package.name>.<type.name>
type TypeHash string

// GetTypeHash returns the TypeHash for a given reflect.Type
func GetTypeHash(t reflect.Type) TypeHash {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return TypeHash(fmt.Sprintf("%s.%s", t.PkgPath(), t.Name()))
}
