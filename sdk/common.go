package sdk

import (
	"reflect"
	"regexp"

	"github.com/go-gorp/gorp"
)

//EncryptFunc  is a common type
type EncryptFunc func(gorp.SqlExecutor, int64, string, string) (string, error)

type IDName struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

// NamePattern  Pattern for project/application/pipeline/group name
const NamePattern = "^[a-zA-Z0-9._-]{1,}$"

// NamePatternRegex  Pattern regexp
var NamePatternRegex = regexp.MustCompile(NamePattern)

// InterfaceSlice cast a untyped slice into a slice of untypes things. It will panic if the parameter is not a slice
func InterfaceSlice(slice interface{}) []interface{} {
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
