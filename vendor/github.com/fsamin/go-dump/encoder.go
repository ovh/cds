package dump

import (
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sort"
	"strings"
)

// Encoder ensures all options to dump an object
type Encoder struct {
	Formatters  []KeyFormatterFunc
	ExtraFields struct {
		Len            bool
		Type           bool
		DetailedStruct bool
		DetailedMap    bool
	}
	writer io.Writer
}

// NewDefaultEncoder instanciate de default encoder
func NewDefaultEncoder(w io.Writer) *Encoder {
	enc := &Encoder{
		Formatters: []KeyFormatterFunc{
			WithDefaultFormatter(),
		},
		writer: w,
	}
	enc.ExtraFields.Len = true
	enc.ExtraFields.Type = true
	return enc
}

// Fdump formats and displays the passed arguments to io.Writer w. It formats exactly the same as Dump.
func (e *Encoder) Fdump(i interface{}) (err error) {
	res, err := e.ToStringMap(i)
	if err != nil {
		return
	}

	keys := []string{}
	for k := range res {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		var err error
		if res[k] == "" {
			_, err = fmt.Fprintf(e.writer, "%s:\n", k)
		} else {
			_, err = fmt.Fprintf(e.writer, "%s: %s\n", k, res[k])
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Sdump returns a string with the passed arguments formatted exactly the same as Dump.
func (e *Encoder) Sdump(i interface{}) (string, error) {
	m, err := e.ToStringMap(i)
	if err != nil {
		return "", err
	}
	res := ""
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		res += fmt.Sprintf("%s: %s\n", k, m[k])
	}
	return res, nil
}

func (e *Encoder) fdumpInterface(w map[string]interface{}, i interface{}, roots []string) error {
	f := valueFromInterface(i)
	if !validAndNotEmpty(f) {
		k := fmt.Sprintf("%s", strings.Join(sliceFormat(roots, e.Formatters), "."))
		w[k] = ""
		return nil
	}
	switch f.Kind() {
	case reflect.Struct:
		if e.ExtraFields.Type {
			nodeType := append(roots, "__Type__")
			nodeTypeFormatted := strings.Join(sliceFormat(nodeType, e.Formatters), ".")
			w[nodeTypeFormatted] = f.Type().Name()
		}
		croots := roots
		if len(roots) == 0 {
			croots = append(roots, f.Type().Name())
		}
		if err := e.fdumpStruct(w, f, croots); err != nil {
			return err
		}
	case reflect.Array, reflect.Slice:
		if e.ExtraFields.Type {
			nodeType := append(roots, "__Type__")
			nodeTypeFormatted := strings.Join(sliceFormat(nodeType, e.Formatters), ".")
			w[nodeTypeFormatted] = "Array"
		}
		if err := e.fDumpArray(w, i, roots); err != nil {
			return err
		}
		return nil
	case reflect.Map:
		if e.ExtraFields.Type {
			nodeType := append(roots, "__Type__")
			nodeTypeFormatted := strings.Join(sliceFormat(nodeType, e.Formatters), ".")
			w[nodeTypeFormatted] = "Map"
		}
		if err := e.fDumpMap(w, i, roots); err != nil {
			return err
		}
		return nil
	default:
		k := fmt.Sprintf("%s", strings.Join(sliceFormat(roots, e.Formatters), "."))
		w[k] = f.Interface()
	}
	return nil
}

func (e *Encoder) fDumpArray(w map[string]interface{}, i interface{}, roots []string) error {
	v := reflect.ValueOf(i)

	if e.ExtraFields.Len {
		nodeLen := append(roots, "__Len__")
		nodeLenFormatted := strings.Join(sliceFormat(nodeLen, e.Formatters), ".")
		w[nodeLenFormatted] = v.Len()
	}

	for i := 0; i < v.Len(); i++ {
		var l string
		//croots := roots
		var croots []string
		if len(roots) > 0 {
			l = roots[len(roots)-1:][0]
			croots = append(roots, fmt.Sprintf("%s%d", l, i))
		} else {
			croots = append(roots, fmt.Sprintf("%s%d", l, i))
		}
		f := v.Index(i)
		if err := e.fdumpInterface(w, f.Interface(), croots); err != nil {
			return err
		}
	}

	return nil
}

func (e *Encoder) fDumpMap(w map[string]interface{}, i interface{}, roots []string) error {
	v := reflect.ValueOf(i)

	keys := v.MapKeys()
	var lenKeys int64
	for _, k := range keys {
		key := fmt.Sprintf("%v", k.Interface())
		if key == "" {
			continue
		}
		lenKeys++
		croots := append(roots, key)
		value := v.MapIndex(k)

		f := valueFromInterface(value.Interface())

		if validAndNotEmpty(f) && f.Type().Kind() == reflect.Struct {
			croots = append(croots, f.Type().Name())
		}

		if err := e.fdumpInterface(w, value.Interface(), croots); err != nil {
			return err
		}
	}

	if e.ExtraFields.Len {
		nodeLen := append(roots, "__Len__")
		nodeLenFormatted := strings.Join(sliceFormat(nodeLen, e.Formatters), ".")
		w[nodeLenFormatted] = lenKeys
	}
	if e.ExtraFields.DetailedMap {
		structKey := fmt.Sprintf("%s", strings.Join(sliceFormat(roots, e.Formatters), "."))
		w[structKey] = i
	}
	return nil
}

func (e *Encoder) fdumpStruct(w map[string]interface{}, s reflect.Value, roots []string) error {
	if e.ExtraFields.DetailedStruct {
		if e.ExtraFields.Len {
			nodeLen := append(roots, "__Len__")
			nodeLenFormatted := strings.Join(sliceFormat(nodeLen, e.Formatters), ".")
			w[nodeLenFormatted] = s.NumField()
		}

		structKey := fmt.Sprintf("%s", strings.Join(sliceFormat(roots, e.Formatters), "."))
		if s.CanInterface() {
			w[structKey] = s.Interface()
		}
	}

	for i := 0; i < s.NumField(); i++ {
		if !s.Field(i).CanInterface() {
			continue
		}
		croots := append(roots, s.Type().Field(i).Name)
		if err := e.fdumpInterface(w, s.Field(i).Interface(), croots); err != nil {
			return err
		}
	}
	return nil
}

// ToStringMap formats the argument as a map[string]string. It formats exactly the same as Dump.
func (e *Encoder) ToStringMap(i interface{}) (res map[string]string, err error) {
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
	ires := map[string]interface{}{}
	if err = e.fdumpInterface(ires, i, nil); err != nil {
		return
	}
	res = map[string]string{}
	for k, v := range ires {
		res[k] = fmt.Sprintf("%v", v)
	}
	return
}

// ToMap dumps argument as a map[string]interface{}
func (e *Encoder) ToMap(i interface{}) (res map[string]interface{}, err error) {
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
	res = map[string]interface{}{}
	if err = e.fdumpInterface(res, i, nil); err != nil {
		return
	}
	return
}
