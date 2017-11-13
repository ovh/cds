package executors

import (
	"bytes"

	dump "github.com/fsamin/go-dump"
)

// Dump dumps v as a map[string]interface{}.
func Dump(v interface{}) (map[string]interface{}, error) {
	w := new(bytes.Buffer)
	e := dump.NewDefaultEncoder(w)

	e.ExtraFields.Len = true
	e.ExtraFields.Type = true
	e.ExtraFields.DetailedStruct = true
	e.ExtraFields.DetailedMap = true
	e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}

	return e.ToMap(v)
}
