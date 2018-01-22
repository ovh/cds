package iffy

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"text/template"
)

type Tester struct {
	t      *testing.T
	r      http.Handler
	Calls  []*Call
	values Values
}

type Call struct {
	Name       string
	Method     string
	QueryStr   string
	Body       interface{}
	headers    http.Header
	respObject interface{}
	checkers   []Checker
}

func (c *Call) ResponseObject(respObject interface{}) *Call {
	c.respObject = respObject
	return c
}

func (c *Call) Headers(h http.Header) *Call {
	c.headers = h
	return c
}

func (c *Call) Checkers(ch ...Checker) *Call {
	c.checkers = ch
	return c
}

type Checker func(r *http.Response, body string, respObject interface{}) error

// Tester

func NewTester(t *testing.T, r http.Handler, calls ...*Call) *Tester {
	return &Tester{
		t:      t,
		r:      r,
		values: make(Values),
	}
}

func (t *Tester) Reset() {
	t.Calls = []*Call{}
}

func (t *Tester) AddCall(name, method, querystr string, body interface{}) *Call {
	c := &Call{
		Name:     name,
		Method:   method,
		QueryStr: querystr,
		Body:     body,
	}
	t.Calls = append(t.Calls, c)
	return c
}

func (t *Tester) Run() {
	for _, c := range t.Calls {
		var body io.Reader

		if c.Body != nil {
			switch c.Body.(type) {
			case string:
				body = bytes.NewBuffer([]byte(t.applyTemplate(c.Body.(string))))
			case []byte:
				body = bytes.NewBuffer(c.Body.([]byte))
			default:
				jsonBtes, err := json.Marshal(c.Body)
				if err != nil {
					t.t.Log(err)
				}
				if jsonBtes != nil {
					body = bytes.NewBuffer(jsonBtes)
				}
			}
		}
		req, err := http.NewRequest(c.Method, t.applyTemplate(c.QueryStr), body)
		if err != nil {
			t.t.Error(err)
			continue
		}
		if c.Body != "" {
			req.Header.Set("content-type", "application/json")
		}
		if c.headers != nil {
			for k := range c.headers {
				v := c.headers.Get(k)
				req.Header.Set(t.applyTemplate(k), t.applyTemplate(v))
			}
		}
		w := httptest.NewRecorder()
		t.r.ServeHTTP(w, req)
		resp := w.Result()
		var respBody string
		if resp.Body != nil {
			rb, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.t.Error(err)
			}
			respBody = string(rb)
			resp.Body.Close()
			if c.respObject != nil {
				err = json.Unmarshal(rb, c.respObject)
				if err != nil {
					t.t.Error(err)
				}
			}
			var retJson map[string]interface{}
			err = json.Unmarshal(rb, &retJson)
			if err == nil {
				t.values[c.Name] = retJson
			}
		}
		for _, checker := range c.checkers {
			err = checker(resp, respBody, c.respObject)
			if err != nil {
				t.t.Errorf("%s: %s", c.Name, err)
			}
		}
	}
}

func (t *Tester) applyTemplate(s string) string {
	b, err := t.values.Apply(s)
	if err != nil {
		t.t.Error(err)
		return ""
	}
	return string(b)
}

type Values map[string]interface{}

func (v Values) Apply(templateStr string) ([]byte, error) {

	var funcMap = template.FuncMap{
		"field": v.fieldTmpl,
		"json":  v.jsonFieldTmpl,
	}

	tmpl, err := template.New("tmpl").Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return nil, err
	}

	b := new(bytes.Buffer)

	err = tmpl.Execute(b, v)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// templating funcs

func (v Values) fieldTmpl(key ...string) (interface{}, error) {
	var i interface{}

	i = map[string]interface{}(v)
	var ok bool

	for _, k := range key {
		switch i.(type) {
		case map[string]interface{}:
			i, ok = i.(map[string]interface{})[k]
			if !ok {
				i = "<no value>"
			}
		case map[string]string:
			i, ok = i.(map[string]string)[k]
			if !ok {
				i = "<no value>"
			}
		default:
			return nil, fmt.Errorf("cannot dereference %T", i)
		}
	}
	return i, nil
}

func (v Values) jsonFieldTmpl(key ...string) (interface{}, error) {
	i, err := v.fieldTmpl(key...)
	if err != nil {
		return nil, err
	}
	marshalled, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return string(marshalled), nil
}

// BUILT IN CHECKERS

func ExpectStatus(st int) Checker {
	return func(r *http.Response, body string, respObject interface{}) error {
		if r.StatusCode != st {
			return fmt.Errorf("Bad status code: expected %d, got %d", st, r.StatusCode)
		}
		return nil
	}
}

func DumpResponse(t *testing.T) Checker {
	return func(r *http.Response, body string, respObject interface{}) error {
		t.Log(body)
		return nil
	}
}

func UnmarshalResponse(i interface{}) Checker {
	return func(r *http.Response, body string, respObject interface{}) error {
		return json.Unmarshal([]byte(body), i)
	}
}

func ExpectJSONFields(fields ...string) Checker {
	return func(r *http.Response, body string, respObject interface{}) error {
		m := map[string]interface{}{}
		err := json.Unmarshal([]byte(body), &m)
		if err != nil {
			return err
		}
		for _, f := range fields {
			if _, ok := m[f]; !ok {
				return fmt.Errorf("Missing expected field '%s'", f)
			}
		}
		return nil
	}
}

func ExpectListLength(length int) Checker {
	return func(r *http.Response, body string, respObject interface{}) error {
		l := []interface{}{}
		err := json.Unmarshal([]byte(body), &l)
		if err != nil {
			return err
		}
		if len(l) != length {
			return fmt.Errorf("Expected a list of length %d, got %d", length, len(l))
		}
		return nil
	}
}

func ExpectListNonEmpty(r *http.Response, body string, respObject interface{}) error {
	l := []interface{}{}
	err := json.Unmarshal([]byte(body), &l)
	if err != nil {
		return err
	}
	if len(l) == 0 {
		return errors.New("Expected a non empty list")
	}
	return nil
}

func ExpectJSONBranch(nodes ...string) Checker {
	return func(r *http.Response, body string, respObject interface{}) error {
		m := map[string]interface{}{}
		err := json.Unmarshal([]byte(body), &m)
		if err != nil {
			return err
		}
		for i, n := range nodes {
			v, ok := m[n]
			if !ok {
				return fmt.Errorf("Missing node '%s'", n)
			}
			if child, ok := v.(map[string]interface{}); ok {
				m = child
			} else if i == len(nodes)-2 {
				// last child is not an object anymore
				// and there's only one more node to check
				// test last child against last provided node
				lastNode := nodes[i+1]
				if v != lastNode {
					return fmt.Errorf("Wrong value: expected '%v', got '%v'", lastNode, v)
				}
				return nil
			}
		}
		return nil
	}
}
