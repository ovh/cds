package test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func NoError(t *testing.T, err error, msg ...interface{}) {
	assert.NoError(t, err)
	if err != nil {
		t.Fatal(msg...)
	}
}

func NotNil(t *testing.T, i interface{}, msg ...interface{}) {
	assert.NotNil(t, i)
	if i == nil {
		t.Fatal(msg...)
	}
}

func NotEmpty(t *testing.T, i interface{}, msg ...interface{}) {
	if !assert.NotEmpty(t, i) {
		t.Fatal(msg...)
	}
}

func interfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("interfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

func arrayContains(array interface{}, s interface{}) bool {
	b := interfaceSlice(array)
	for _, i := range b {
		if reflect.DeepEqual(i, s) {
			return true
		}
	}
	return false
}

func EqualValuesWithoutOrder(t *testing.T, a, b interface{}, msgAndArgs ...interface{}) {
	s1 := interfaceSlice(a)
	s2 := interfaceSlice(b)

	for _, x := range s1 {
		if !arrayContains(s2, x) {
			assert.Fail(t, "EqualValuesWithoutOrder failed", msgAndArgs...)
		}
	}

	if t.Failed() {
		return
	}

	for _, x := range s2 {
		if !arrayContains(s1, x) {
			assert.Fail(t, "EqualValuesWithoutOrder failed", msgAndArgs...)
		}
	}
}
