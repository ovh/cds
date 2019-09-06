package sdk

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestReflectFieldByTag(t *testing.T) {
	var i1 = struct {
		Field1 string `tag:"value"`
	}{
		Field1: "blabla",
	}

	f1 := ReflectFieldByTag(&i1, "tag", "value")
	require.NotNil(t, f1)
	s, ok := f1.(*string)
	require.True(t, ok)
	require.Equal(t, i1.Field1, *s)

	var i2 = struct {
		Field1 struct {
			Field2 string `tag:"field2"`
		} `tag:"field1"`
	}{}
	i2.Field1.Field2 = "blabla"

	f2 := ReflectFieldByTag(&i2, "tag", "field1.field2")
	require.NotNil(t, f2)
	s, ok = f2.(*string)
	require.True(t, ok)
	require.Equal(t, i2.Field1.Field2, *s)

	val := reflect.NewAt(reflect.ValueOf(f2).Elem().Type(), unsafe.Pointer(reflect.ValueOf(f2).Pointer()))
	newValue := "bloublou"
	val.Elem().Set(reflect.ValueOf(newValue))

	require.Equal(t, newValue, i2.Field1.Field2)

}
