package sdk

// This functions come from https://github.com/Masterminds/sprig
// Copyright (C) 2013 Masterminds
// Masterminds/sprig is licensed under the MIT License

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	util "github.com/aokoli/goutils"
	"github.com/huandu/xstrings"
)

var interpolateHelperFuncs template.FuncMap

func init() {
	interpolateHelperFuncs = template.FuncMap{
		"abbrev":     abbrev,
		"abbrevboth": abbrevboth,
		"trunc":      trunc,
		"trim":       strings.TrimSpace,
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      strings.Title,
		"untitle":    untitle,
		"substr":     substring,
		// Switch order so that "foo" | repeat 5
		"repeat": func(count int, str string) string { return strings.Repeat(str, count) },
		// Deprecated: Use trimAll.
		"trimall": func(a, b string) string { return strings.Trim(b, a) },
		// Switch order so that "$foo" | trimall "$"
		"trimAll":      func(a, b string) string { return strings.Trim(b, a) },
		"trimSuffix":   func(a, b string) string { return strings.TrimSuffix(b, a) },
		"trimPrefix":   func(a, b string) string { return strings.TrimPrefix(b, a) },
		"nospace":      util.DeleteWhiteSpace,
		"initials":     initials,
		"randAlphaNum": randAlphaNumeric,
		"randAlpha":    randAlpha,
		"randAscii":    randAscii,
		"randNumeric":  randNumeric,
		"swapcase":     util.SwapCase,
		"shuffle":      xstrings.Shuffle,
		"snakecase":    xstrings.ToSnakeCase,
		"camelcase":    xstrings.ToCamelCase,
		"quote":        quote,
		"squote":       squote,
		"indent":       indent,
		"nindent":      nindent,
		"replace":      replace,
		"plural":       plural,
		"toString":     strval,
		"default":      dfault,
		"empty":        empty,
		"coalesce":     coalesce,
		"toJson":       toJson,
		"toPrettyJson": toPrettyJson,
		"b64enc":       base64encode,
		"b64dec":       base64decode,
		"escape":       escape,
	}
}

// dfault checks whether `given` is set, and returns default if not set.
//
// This returns `d` if `given` appears not to be set, and `given` otherwise.
//
// For numeric types 0 is unset.
// For strings, maps, arrays, and slices, len() = 0 is considered unset.
// For bool, false is unset.
// Structs are never considered unset.
//
// For everything else, including pointers, a nil value is unset.
func dfault(d interface{}, given ...interface{}) interface{} {

	if empty(given) || empty(given[0]) {
		return d
	}
	return given[0]
}

// empty returns true if the given value has the zero value for its type.
func empty(given interface{}) bool {
	g := reflect.ValueOf(given)
	if !g.IsValid() {
		return true
	}

	// Basically adapted from text/template.isTrue
	switch g.Kind() {
	default:
		return g.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		return g.Len() == 0
	case reflect.Bool:
		return g.Bool() == false
	case reflect.Complex64, reflect.Complex128:
		return g.Complex() == 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return g.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return g.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return g.Float() == 0
	case reflect.Struct:
		return false
	}
	return true
}

// coalesce returns the first non-empty value.
func coalesce(v ...interface{}) interface{} {
	for _, val := range v {
		if !empty(val) {
			return val
		}
	}
	return nil
}

// toJson encodes an item into a JSON string
func toJson(v interface{}) string {
	output, _ := json.Marshal(v)
	return string(output)
}

// toPrettyJson encodes an item into a pretty (indented) JSON string
func toPrettyJson(v interface{}) string {
	output, _ := json.MarshalIndent(v, "", "  ")
	return string(output)
}

func base64encode(v string) string {
	return base64.StdEncoding.EncodeToString([]byte(v))
}

func base64decode(v string) string {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

func abbrev(width int, s string) string {
	if width < 4 {
		return s
	}
	r, _ := util.Abbreviate(s, width)
	return r
}

func abbrevboth(left, right int, s string) string {
	if right < 4 || left > 0 && right < 7 {
		return s
	}
	r, _ := util.AbbreviateFull(s, left, right)
	return r
}
func initials(s string) string {
	// Wrap this just to eliminate the var args, which templates don't do well.
	return util.Initials(s)
}

func randAlphaNumeric(count int) string {
	// It is not possible, it appears, to actually generate an error here.
	r, _ := util.RandomAlphaNumeric(count)
	return r
}

func randAlpha(count int) string {
	r, _ := util.RandomAlphabetic(count)
	return r
}

func randAscii(count int) string {
	r, _ := util.RandomAscii(count)
	return r
}

func randNumeric(count int) string {
	r, _ := util.RandomNumeric(count)
	return r
}

func untitle(str string) string {
	return util.Uncapitalize(str)
}

func quote(str ...interface{}) string {
	out := make([]string, len(str))
	for i, s := range str {
		out[i] = fmt.Sprintf("%q", strval(s))
	}
	return strings.Join(out, " ")
}

func squote(str ...interface{}) string {
	out := make([]string, len(str))
	for i, s := range str {
		out[i] = fmt.Sprintf("'%v'", s)
	}
	return strings.Join(out, " ")
}

func cat(v ...interface{}) string {
	r := strings.TrimSpace(strings.Repeat("%v ", len(v)))
	return fmt.Sprintf(r, v...)
}

func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	return pad + strings.Replace(v, "\n", "\n"+pad, -1)
}

func nindent(spaces int, v string) string {
	return "\n" + indent(spaces, v)
}

func replace(old, new, src string) string {
	return strings.Replace(src, old, new, -1)
}

func plural(one, many string, count int) string {
	if count == 1 {
		return one
	}
	return many
}

func strslice(v interface{}) []string {
	switch v := v.(type) {
	case []string:
		return v
	case []interface{}:
		l := len(v)
		b := make([]string, l)
		for i := 0; i < l; i++ {
			b[i] = strval(v[i])
		}
		return b
	default:
		val := reflect.ValueOf(v)
		switch val.Kind() {
		case reflect.Array, reflect.Slice:
			l := val.Len()
			b := make([]string, l)
			for i := 0; i < l; i++ {
				b[i] = strval(val.Index(i).Interface())
			}
			return b
		default:
			return []string{strval(v)}
		}
	}
}

func strval(v interface{}) string {
	switch v := v.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case error:
		return v.Error()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func trunc(c int, s string) string {
	if len(s) <= c {
		return s
	}
	return s[0:c]
}

func join(sep string, v interface{}) string {
	return strings.Join(strslice(v), sep)
}

func split(sep, orig string) map[string]string {
	parts := strings.Split(orig, sep)
	res := make(map[string]string, len(parts))
	for i, v := range parts {
		res["_"+strconv.Itoa(i)] = v
	}
	return res
}

// substring creates a substring of the given string.
//
// If start is < 0, this calls string[:length].
//
// If start is >= 0 and length < 0, this calls string[start:]
//
// Otherwise, this calls string[start, length].
func substring(start, length int, s string) string {
	if start < 0 {
		return s[:length]
	}
	if length < 0 {
		return s[start:]
	}
	return s[start:length]
}

func escape(s string) string {
	s1 := strings.Replace(s, "_", "-", -1)
	s1 = strings.Replace(s1, "/", "-", -1)
	s1 = strings.Replace(s1, ".", "-", -1)
	return s1
}
