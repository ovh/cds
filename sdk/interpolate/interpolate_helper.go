package interpolate

// This functions come from https://github.com/Masterminds/sprig
// Copyright (C) 2013 Masterminds
// Masterminds/sprig is licensed under the MIT License

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	util "github.com/aokoli/goutils"
	"github.com/huandu/xstrings"
	"github.com/spf13/cast"
)

// InterpolateHelperFuncs is a list of funcs that can be used in go templates.
var InterpolateHelperFuncs template.FuncMap

func init() {
	InterpolateHelperFuncs = wrapHelpers(template.FuncMap{
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
		"randASCII":    randASCII,
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
		"toJSON":       toJSON,
		"toPrettyJSON": toPrettyJSON,
		"b64enc":       base64encode,
		"b64dec":       base64decode,
		"escape":       escape,
		"stringQuote":  stringQuote,
		"add": func(i ...interface{}) int64 {
			var a int64 = 0
			for _, b := range i {
				a += toInt64(b)
			}
			return a
		},
		"sub": func(a, b interface{}) int64 { return toInt64(a) - toInt64(b) },
		"mul": func(a interface{}, v ...interface{}) int64 {
			val := toInt64(a)
			for _, b := range v {
				val = val * toInt64(b)
			}
			return val
		},
		"div":       func(a, b interface{}) int64 { return toInt64(a) / toInt64(b) },
		"mod":       func(a, b interface{}) int64 { return toInt64(a) % toInt64(b) },
		"ternary":   ternary,
		"urlencode": func(s string) string { return url.QueryEscape(s) },
		"dirname":   func(s string) string { return path.Dir(s) },
		"basename":  func(s string) string { return path.Base(s) },
	})
}

// wrapHelpers to handle usage of val struct in interpolate.Do
func wrapHelpers(fs template.FuncMap) template.FuncMap {
	wrappedHelpers := make(template.FuncMap, len(fs))
	for key, helper := range fs {
		helperV := reflect.ValueOf(helper)

		// ignore if current helper is not a func
		if helperV.Kind() != reflect.Func {
			continue
		}

		helperT := helperV.Type()
		paramsCount := helperT.NumIn()
		paramsTypes := make([]string, paramsCount)
		for i := 0; i < paramsCount; i++ {
			paramsTypes[i] = helperT.In(i).Name()
		}

		// create the wrapper func
		wrappedHelpers[key] = func(ps ...interface{}) interface{} {
			// if the helper func need more params than ps length, throw an error
			if len(ps) < paramsCount {
				// panic will be catched be text/template executor
				panic(fmt.Sprintf("missing params (expected: %s)", strings.Join(paramsTypes, ", ")))
			}

			// for all helper's params, forward values from wrapper
			values := make([]reflect.Value, len(ps))
			for i := 0; i < len(ps); i++ {
				if value, ok := ps[i].(*val); ok {
					// if the value is a pointer to val, we should return its internal value
					values[i] = reflect.ValueOf((*value)["_"])
				} else if value, ok := ps[i].(val); ok {
					// if the value is a val, we should return its internal value
					values[i] = reflect.ValueOf(value["_"])
				} else if v := reflect.ValueOf(ps[i]); v.IsValid() {
					// for all params that are not val (string, integer...) use it directly
					values[i] = v
				} else {
					// if the value is not valid (means that given value is nil with unknown type), convert to nil void pointer
					var v *void
					values[i] = reflect.ValueOf(v)
				}
			}

			results := helperV.Call(values)
			if len(results) == 0 {
				return nil
			}

			return results[0].Interface()
		}
	}
	return wrappedHelpers
}

// Switch order so that "$foo" | trimall "$"
func dfault(valfault ...interface{}) string {
	var castToString = func(i interface{}) string {
		s, _ := i.(string)
		return s
	}

	if len(valfault) == 0 {
		return ""
	}
	if len(valfault) == 1 {
		s := castToString(valfault[0])
		return s
	}
	for i := len(valfault) - 1; i >= 0; i-- {
		if s := castToString(valfault[i]); s != "" {
			return s
		}
	}

	return ""
}

// empty returns true if the given value has the zero value for its type.
func empty(given interface{}) bool {
	g := reflect.ValueOf(given)
	if !g.IsValid() {
		return true
	}

	// Basically adapted from text/template.isTrue
	switch g.Kind() {
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
	default:
		return g.IsNil()
	}
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

// toJSON encodes an item into a JSON string
func toJSON(v interface{}) string {
	output, _ := json.Marshal(v)
	return string(output)
}

// toPrettyJSON encodes an item into a pretty (indented) JSON string
func toPrettyJSON(v interface{}) string {
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

func randASCII(count int) string {
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

func stringQuote(s string) string {
	x := strconv.Quote(s)
	x = strings.TrimPrefix(x, `"`)
	x = strings.TrimSuffix(x, `"`)
	return x
}

func ternary(v, v2, a interface{}) interface{} {
	if cast.ToBool(a) {
		return v
	}
	return v2
}

// toInt64 converts integer types to 64-bit integers
func toInt64(v interface{}) int64 {
	return cast.ToInt64(v)
}
