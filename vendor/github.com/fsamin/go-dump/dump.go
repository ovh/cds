package dump

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"runtime"
	"strings"
)

// Dump displays the passed parameter properties to standard out such as complete types and all
// pointer addresses used to indirect to the final value.
// See Fdump if you would prefer dumping to an arbitrary io.Writer or Sdump to
// get the formatted result as a string.
func Dump(i interface{}) error {
	return FDump(os.Stdout, i)
}

// FDump formats and displays the passed arguments to io.Writer w. It formats exactly the same as Dump.
func FDump(w io.Writer, i interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
			buf := make([]byte, 1<<16)
			runtime.Stack(buf, true)
			fmt.Printf("%s", buf)
		}
	}()
	return fDumpStruct(w, i)
}

// Sdump returns a string with the passed arguments formatted exactly the same as Dump.
func Sdump(i interface{}) (string, error) {
	var buf bytes.Buffer
	if err := FDump(&buf, i); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func fDumpStruct(w io.Writer, i interface{}, roots ...string) error {
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
		for i := 0; i < s.NumField(); i++ {
			f := s.Field(i)
			if f.Kind() == reflect.Ptr {
				f = f.Elem()
			}
			switch f.Kind() {
			case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
				var data interface{}
				if validAndNotEmpty(f) {
					data = f.Interface()
				} else {
					data = nil
				}
				if err := fDumpStruct(w, data, append(roots, s.Type().Field(i).Name)...); err != nil {
					return err
				}
			default:
				var data interface{}
				if validAndNotEmpty(f) {
					data = f.Interface()
					res := fmt.Sprintf("%s.%s: %v\n", strings.Join(roots, "."), s.Type().Field(i).Name, data)
					if _, err := w.Write([]byte(res)); err != nil {
						return err
					}
				}
			}
		}
	case reflect.Array, reflect.Slice:
		if err := fDumpArray(w, i, roots...); err != nil {
			return err
		}
		return nil
	case reflect.Map:
		if err := fDumpMap(w, i, roots...); err != nil {
			return err
		}
		return nil
	default:
		roots = append(roots, s.Type().Name())
		var data interface{}
		if validAndNotEmpty(s) {
			data = s.Interface()
			res := fmt.Sprintf("%s.%s: %v\n", strings.Join(roots, "."), s.Type().Name(), data)
			if _, err := w.Write([]byte(res)); err != nil {
				return err
			}
		}
		return nil
	}

	return nil
}

func fDumpArray(w io.Writer, i interface{}, roots ...string) error {
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
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			var data interface{}
			if f.IsValid() {
				data = f.Interface()
			} else {
				data = nil
			}
			if err := fDumpStruct(w, data, croots...); err != nil {
				return err
			}
		default:
			var data interface{}
			if f.IsValid() {
				data = f.Interface()
				res := fmt.Sprintf("%s: %v\n", strings.Join(croots, "."), data)
				if _, err := w.Write([]byte(res)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func fDumpMap(w io.Writer, i interface{}, roots ...string) error {
	v := reflect.ValueOf(i)
	keys := v.MapKeys()
	//TODO  should manager map of pointer
	for _, k := range keys {
		key := fmt.Sprintf("%v", k.Interface())
		key = strings.Replace(key, " ", "_", -1)
		key = strings.Replace(key, "/", "_", -1)
		roots := append(roots, key)
		switch v.MapIndex(k).Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			if err := fDumpStruct(w, v.MapIndex(k).Interface(), roots...); err != nil {
				return err
			}
		default:
			res := fmt.Sprintf("%s: %v\n", strings.Join(roots, "."), v.MapIndex(k).Interface())
			if _, err := w.Write([]byte(res)); err != nil {
				return err
			}
		}
	}
	return nil
}

type mapWriter struct {
	data map[string]string
}

func (m *mapWriter) Write(p []byte) (int, error) {
	if m.data == nil {
		m.data = map[string]string{}
	}
	tuple := strings.SplitN(string(p), ":", 2)
	if len(tuple) != 2 {
		return 0, errors.New("malformatted bytes")
	}
	tuple[1] = strings.Replace(tuple[1], "\n", "", -1)
	m.data[strings.TrimSpace(tuple[0])] = strings.TrimSpace(tuple[1])
	return len(p), nil
}

// ToMap format passed parameter as a map[string]string. It formats exactly the same as Dump.
func ToMap(i interface{}) (map[string]string, error) {
	m := mapWriter{}
	err := FDump(&m, i)
	return m.data, err
}

func validAndNotEmpty(v reflect.Value) bool {
	if v.IsValid() {
		if v.Kind() == reflect.String {
			return v.String() != ""
		}
		return true
	}
	return false
}
