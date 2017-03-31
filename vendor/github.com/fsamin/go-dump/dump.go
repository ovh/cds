package dump

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// KeyFormatterFunc is a type for key formatting
type KeyFormatterFunc func(s string) string

// WithLowerCaseFormatter formats keys in lowercase
func WithLowerCaseFormatter() KeyFormatterFunc {
	return func(s string) string {
		return strings.ToLower(s)
	}
}

// WithDefaultLowerCaseFormatter formats keys in lowercase and apply default formatting
func WithDefaultLowerCaseFormatter() KeyFormatterFunc {
	f := WithDefaultFormatter()
	return func(s string) string {
		return strings.ToLower(f(s))
	}
}

// WithDefaultFormatter is the default formatter
func WithDefaultFormatter() KeyFormatterFunc {
	return func(s string) string {
		s = strings.Replace(s, " ", "_", -1)
		s = strings.Replace(s, "/", "_", -1)
		s = strings.Replace(s, ":", "_", -1)
		return s
	}
}

// NoFormatter doesn't do anything, so to be sure to avoid keys formatting, use only this formatter
func NoFormatter() KeyFormatterFunc {
	return func(s string) string {
		return s
	}
}

// Dump displays the passed parameter properties to standard out such as complete types and all
// pointer addresses used to indirect to the final value.
// See Fdump if you would prefer dumping to an arbitrary io.Writer or Sdump to
// get the formatted result as a string.
func Dump(i interface{}, formatters ...KeyFormatterFunc) error {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	return Fdump(os.Stdout, i, formatters...)
}

// Fdump formats and displays the passed arguments to io.Writer w. It formats exactly the same as Dump.
func Fdump(w io.Writer, i interface{}, formatters ...KeyFormatterFunc) (err error) {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}

	res, err := ToMap(i, formatters...)
	if err != nil {
		return
	}

	keys := []string{}
	for k := range res {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, err := fmt.Fprintf(w, "%s: %s\n", k, res[k])
		if err != nil {
			return err
		}
	}
	return nil
}

// Sdump returns a string with the passed arguments formatted exactly the same as Dump.
func Sdump(i interface{}, formatters ...KeyFormatterFunc) (string, error) {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	m, err := ToMap(i, formatters...)
	if err != nil {
		return "", err
	}
	res := ""
	for k, v := range m {
		res += fmt.Sprintf("%s: %s\n", k, v)
	}
	return res, nil
}

func fdumpStructField(w map[string]string, s reflect.Value, roots []string, formatters ...KeyFormatterFunc) error {
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}
		switch f.Kind() {
		case reflect.Struct:
			if validAndNotEmpty(f) {
				if err := fdumpStructField(w, f, append(roots, s.Type().Field(i).Name), formatters...); err != nil {
					return err
				}
			}
		case reflect.Array, reflect.Slice, reflect.Map:
			var data interface{}
			if validAndNotEmpty(f) {
				data = f.Interface()
			} else {
				data = nil
			}
			if err := fdumpStruct(w, data, append(roots, s.Type().Field(i).Name), formatters...); err != nil {
				return err
			}
		default:
			var data interface{}
			if validAndNotEmpty(f) {
				data = f.Interface()
				if f.Kind() == reflect.Interface {
					im := map[string]interface{}{}
					if mapstructure.Decode(data, im) != nil {
						if err := fDumpMap(w, im, append(roots, s.Type().Field(i).Name), formatters...); err != nil {
							return err
						}
						continue
					}
					am, ok := data.([]interface{})
					if ok {
						if err := fDumpArray(w, am, append(roots, s.Type().Field(i).Name), formatters...); err != nil {
							return err
						}
						continue
					}
				}
				k := fmt.Sprintf("%s.%s", strings.Join(sliceFormat(roots, formatters), "."), format(s.Type().Field(i).Name, formatters))
				w[k] = fmt.Sprintf("%v", data)
			}
		}
	}
	return nil
}

func fdumpStruct(w map[string]string, i interface{}, roots []string, formatters ...KeyFormatterFunc) error {
	var s reflect.Value
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		s = reflect.ValueOf(i).Elem()
	} else {
		s = reflect.ValueOf(i)
	}

	if !validAndNotEmpty(s) {
		return nil
	}
	switch s.Kind() {
	case reflect.Struct:
		roots = append(roots, s.Type().Name())
		if err := fdumpStructField(w, s, roots, formatters...); err != nil {
			return err
		}
	case reflect.Array, reflect.Slice:
		if err := fDumpArray(w, i, roots, formatters...); err != nil {
			return err
		}
		return nil
	case reflect.Map:
		if err := fDumpMap(w, i, roots, formatters...); err != nil {
			return err
		}
		return nil
	default:
		roots = append(roots, s.Type().Name())
		var data interface{}
		if validAndNotEmpty(s) {
			data = s.Interface()
			if s.Kind() == reflect.Interface {
				im := map[string]interface{}{}
				if mapstructure.Decode(data, im) != nil {
					return fDumpMap(w, im, roots, formatters...)
				}
				am, ok := data.([]interface{})
				if ok {
					return fDumpArray(w, am, roots, formatters...)
				}
			}
			k := fmt.Sprintf("%s.%s", strings.Join(sliceFormat(roots, formatters), "."), format(s.Type().Name(), formatters))
			w[k] = fmt.Sprintf("%v", data)
		}
		return nil
	}

	return nil
}

func fDumpArray(w map[string]string, i interface{}, roots []string, formatters ...KeyFormatterFunc) error {
	v := reflect.ValueOf(i)
	for i := 0; i < v.Len(); i++ {
		var l string
		var croots []string
		if len(roots) > 0 {
			l = roots[len(roots)-1:][0]
			croots = roots[:len(roots)-1]
		}
		croots = append(roots, fmt.Sprintf("%s%d", l, i))
		f := v.Index(i)
		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}
		switch f.Kind() {
		case reflect.Struct:
			if f.IsValid() {
				if err := fdumpStructField(w, f, croots, formatters...); err != nil {
					return err
				}
			}
		case reflect.Array, reflect.Slice, reflect.Map:
			var data interface{}
			if f.IsValid() {
				data = f.Interface()
			} else {
				data = nil
			}
			if err := fdumpStruct(w, data, croots, formatters...); err != nil {
				return err
			}
		default:
			var data interface{}
			if validAndNotEmpty(f) {
				data = f.Interface()
				if f.Kind() == reflect.Interface {
					im := map[string]interface{}{}
					if mapstructure.Decode(data, im) != nil {
						if err := fDumpMap(w, im, croots, formatters...); err != nil {
							return err
						}
						continue
					}
					am, ok := data.([]interface{})
					if ok {
						if err := fDumpArray(w, am, croots, formatters...); err != nil {
							return err
						}
						continue
					}
				}

				k := strings.Join(sliceFormat(croots, formatters), ".")
				w[k] = fmt.Sprintf("%v", data)
			}
		}
	}

	return nil
}

func fDumpMap(w map[string]string, i interface{}, roots []string, formatters ...KeyFormatterFunc) error {
	v := reflect.ValueOf(i)
	keys := v.MapKeys()
	for _, k := range keys {
		key := fmt.Sprintf("%v", k.Interface())
		roots := append(roots, key)
		value := v.MapIndex(k)
		if value.Kind() == reflect.Ptr {
			value = value.Elem()
		}
		switch v.MapIndex(k).Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			if err := fdumpStruct(w, value.Interface(), roots, formatters...); err != nil {
				return err
			}
		default:
			if value.Kind() == reflect.Interface {
				im := map[string]interface{}{}
				if mapstructure.Decode(value.Interface(), im) != nil {
					if err := fDumpMap(w, im, roots, formatters...); err != nil {
						return err
					}
					continue
				}
				am, ok := value.Interface().([]interface{})
				if ok {
					if err := fDumpArray(w, am, roots, formatters...); err != nil {
						return err
					}
					continue
				}
			}
			k := strings.Join(sliceFormat(roots, formatters), ".")
			w[k] = fmt.Sprintf("%v", value.Interface())
		}
	}
	return nil
}

// ToMap format passed parameter as a map[string]string. It formats exactly the same as Dump.
func ToMap(i interface{}, formatters ...KeyFormatterFunc) (res map[string]string, err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
		}
	}()
	res = map[string]string{}
	if err = fdumpStruct(res, i, nil, formatters...); err != nil {
		return
	}
	return
}

func validAndNotEmpty(v reflect.Value) bool {
	if v.IsValid() && v.CanInterface() {
		if v.Kind() == reflect.String {
			return v.String() != ""
		}
		return true
	}
	return false
}

func sliceFormat(s []string, formatters []KeyFormatterFunc) []string {
	for i := range s {
		s[i] = format(s[i], formatters)
	}
	return s
}

func format(s string, formatters []KeyFormatterFunc) string {
	for _, f := range formatters {
		s = f(s)
	}
	return s
}

// MustSdump is a helper that wraps a call to a function returning (string, error)
// and panics if the error is non-nil.
func MustSdump(i interface{}, formatters ...KeyFormatterFunc) string {
	s, err := Sdump(i, formatters...)
	if err != nil {
		panic(err)
	}
	return s
}
