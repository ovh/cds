package tonic_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/loopfz/gadgeto/tonic"
)

var r http.Handler

func errorHook(c *gin.Context, e error) (int, interface{}) {

	if _, ok := e.(tonic.InputError); ok {
		return 400, e.Error()
	}
	return 500, e.Error()
}

func TestMain(m *testing.M) {

	tonic.SetErrorHook(errorHook)

	g := gin.Default()
	g.GET("/simple", tonic.Handler(simpleHandler, 200))
	g.GET("/scalar", tonic.Handler(scalarHandler, 200))
	g.GET("/error", tonic.Handler(errorHandler, 200))
	g.GET("/path/:param", tonic.Handler(pathHandler, 200))
	g.GET("/query", tonic.Handler(queryHandler, 200))
	g.POST("/body", tonic.Handler(bodyHandler, 200))

	r = g

	m.Run()
}

func TestSimple(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("simple", "GET", "/simple", "").Checkers(iffy.ExpectStatus(200), expectEmptyBody)
	tester.AddCall("simple", "GET", "/simple/", "").Checkers(iffy.ExpectStatus(301))
	tester.AddCall("simple", "GET", "/simple?", "").Checkers(iffy.ExpectStatus(200))
	tester.AddCall("simple", "GET", "/simple", "{}").Checkers(iffy.ExpectStatus(200))
	tester.AddCall("simple", "GET", "/simple?param=useless", "{}").Checkers(iffy.ExpectStatus(200))

	tester.AddCall("scalar", "GET", "/scalar", "").Checkers(iffy.ExpectStatus(200))

	tester.Run()
}

func TestError(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("error", "GET", "/error", "").Checkers(iffy.ExpectStatus(500))

	tester.Run()
}

func TestPathQuery(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("path", "GET", "/path/foo", "").Checkers(iffy.ExpectStatus(200), expectString("param", "foo"))

	tester.AddCall("query", "GET", "/query?param=foo", "").Checkers(iffy.ExpectStatus(200), expectString("param", "foo"))
	tester.AddCall("query", "GET", "/query?param=foo&param=bar", "").Checkers(iffy.ExpectStatus(400))
	tester.AddCall("query", "GET", "/query?param=", "").Checkers(iffy.ExpectStatus(200))
	tester.AddCall("query", "GET", "/query", "").Checkers(iffy.ExpectStatus(400))
	tester.AddCall("query", "GET", "/query?param=foo&param-optional=bar", "").Checkers(iffy.ExpectStatus(200), expectString("param-optional", "bar"))
	tester.AddCall("query", "GET", "/query?param=foo&param-int=42", "").Checkers(iffy.ExpectStatus(200), expectInt("param-int", 42))
	tester.AddCall("query", "GET", "/query?param=foo&params=foo&params=bar", "").Checkers(iffy.ExpectStatus(200), expectStringArr("params", "foo", "bar"))
	tester.AddCall("query", "GET", "/query?param=foo&param-bool=true", "").Checkers(iffy.ExpectStatus(200), expectBool("param-bool", true))
	tester.AddCall("query", "GET", "/query?param=foo&param-default=bla", "").Checkers(iffy.ExpectStatus(200), expectString("param-default", "bla"))
	tester.AddCall("query", "GET", "/query?param=foo", "").Checkers(iffy.ExpectStatus(200), expectString("param-default", "default"))
	tester.AddCall("query", "GET", "/query?param=foo&param-ptr=bar", "").Checkers(iffy.ExpectStatus(200), expectString("param-ptr", "bar"))
	tester.AddCall("query", "GET", "/query?param=foo&param-embed=bar", "").Checkers(iffy.ExpectStatus(200), expectString("param-embed", "bar"))

	now, _ := time.Time{}.Add(87 * time.Hour).MarshalText()

	tester.AddCall("query", "GET", fmt.Sprintf("/query?param=foo&param-complex=%s", now), "").Checkers(iffy.ExpectStatus(200), expectString("param-complex", string(now)))

	tester.Run()
}

func TestBody(t *testing.T) {

	tester := iffy.NewTester(t, r)

	tester.AddCall("body", "POST", "/body", `{"param": "foo"}`).Checkers(iffy.ExpectStatus(200), expectString("param", "foo"))
	tester.AddCall("body", "POST", "/body", `{}`).Checkers(iffy.ExpectStatus(400))
	tester.AddCall("body", "POST", "/body", `{"param": ""}`).Checkers(iffy.ExpectStatus(400))
	tester.AddCall("body", "POST", "/body", `{"param": "foo", "param-optional": "bar"}`).Checkers(iffy.ExpectStatus(200), expectString("param-optional", "bar"))

	tester.Run()
}

func errorHandler(c *gin.Context) error {
	return errors.New("error")
}

func simpleHandler(c *gin.Context) error {
	return nil
}

func scalarHandler(c *gin.Context) (string, error) {
	return "", nil
}

type pathIn struct {
	Param string `path:"param" json:"param"`
}

func pathHandler(c *gin.Context, in *pathIn) (*pathIn, error) {
	return in, nil
}

type queryIn struct {
	Param         string    `query:"param, required" json:"param"`
	ParamOptional string    `query:"param-optional" json:"param-optional"`
	Params        []string  `query:"params" json:"params"`
	ParamInt      int       `query:"param-int" json:"param-int"`
	ParamBool     bool      `query:"param-bool" json:"param-bool"`
	ParamDefault  string    `query:"param-default, default=default" json:"param-default"`
	ParamPtr      *string   `query:"param-ptr" json:"param-ptr"`
	ParamComplex  time.Time `query:"param-complex" json:"param-complex"`
	*DoubleEmbedded
}

type Embedded struct {
	ParamEmbed string `query:"param-embed" json:"param-embed"`
}

type DoubleEmbedded struct {
	Embedded
}

func queryHandler(c *gin.Context, in *queryIn) (*queryIn, error) {
	return in, nil
}

type bodyIn struct {
	Param         string `json:"param" binding:"required"`
	ParamOptional string `json:"param-optional"`
}

func bodyHandler(c *gin.Context, in *bodyIn) (*bodyIn, error) {
	return in, nil
}

func expectEmptyBody(r *http.Response, body string, obj interface{}) error {
	if len(body) != 0 {
		return fmt.Errorf("Body '%s' should be empty", body)
	}
	return nil
}

func expectString(paramName, value string) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		var i map[string]interface{}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		s, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		if s != value {
			return fmt.Errorf("%s: expected %s got %s", paramName, value, s)
		}
		return nil
	}
}

func expectBool(paramName string, value bool) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		i := map[string]interface{}{paramName: 0}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		v, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		vb, ok := v.(bool)
		if !ok {
			return fmt.Errorf("%s not a number", paramName)
		}
		if vb != value {
			return fmt.Errorf("%s: expected %v got %v", paramName, value, vb)
		}
		return nil
	}
}

func expectStringArr(paramName string, value ...string) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		var i map[string]interface{}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		s, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		sArr, ok := s.([]interface{})
		if !ok {
			return fmt.Errorf("%s not a string arr", paramName)
		}
		for n := 0; n < len(value); n++ {
			if n >= len(sArr) {
				return fmt.Errorf("%s too short", paramName)
			}
			if sArr[n] != value[n] {
				return fmt.Errorf("%s: %s does not match", paramName, sArr[n])
			}
		}
		return nil
	}
}

func expectInt(paramName string, value int) func(*http.Response, string, interface{}) error {

	return func(r *http.Response, body string, obj interface{}) error {

		i := map[string]interface{}{paramName: 0}

		err := json.Unmarshal([]byte(body), &i)
		if err != nil {
			return err
		}
		v, ok := i[paramName]
		if !ok {
			return fmt.Errorf("%s missing", paramName)
		}
		vf, ok := v.(float64)
		if !ok {
			return fmt.Errorf("%s not a number", paramName)
		}
		vInt := int(vf)
		if vInt != value {
			return fmt.Errorf("%s: expected %v got %v", paramName, value, vInt)
		}
		return nil
	}
}
