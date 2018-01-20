package swagger

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Transforms a go type to swagger type/format
// it is incomplete for now
func GoTypeToSwagger(t reflect.Type) (string, string, string) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {

		if "Time" == t.Name() && t.PkgPath() == "time" {
			return "string", "dateTime", ""
		}

		return "", "", ModelName(t)
	}

	switch t.Kind() {
	case reflect.Int64, reflect.Uint64:
		return "integer", "int64", ""
	case reflect.Int, reflect.Int32, reflect.Uint32, reflect.Uint8, reflect.Uint16:
		return "integer", "int32", ""
	case reflect.Float64:
		return "number", "float", ""
	case reflect.Bool:
		return "boolean", "", ""
	case reflect.String:
		return "string", "", ""
	}
	fmt.Fprintf(os.Stderr, "unhandled type: patch me -> %+v\n", t)
	return "", "", ""
}

func ModelName(t reflect.Type) string {
	// ucFirst to make types public in go, could be handled another way
	modelName := ucFirst(t.Name())

	// we need to avoid collisions, but as we can't afford to break existing code when we detect one
	// we'll always prefix types...

	pkg := ""
	tmpSplit := strings.Split(t.PkgPath(), "/")
	pkg = tmpSplit[len(tmpSplit)-1]

	if strings.ToLower(pkg) != strings.ToLower(modelName) && pkg != "handler" && strings.Index(strings.ToLower(modelName), strings.ToLower(pkg)) != 0 {
		modelName = ucFirst(pkg) + modelName
	}
	return modelName
}

func ucFirst(s string) string {
	if s == "" {
		return ""
	}
	r, n := utf8.DecodeRuneInString(strings.ToLower(s))
	return string(unicode.ToUpper(r)) + s[n:]
}
