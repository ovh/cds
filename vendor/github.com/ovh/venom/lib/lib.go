package venom

import (
	"testing"

	"encoding/json"

	"github.com/ovh/venom"
	"github.com/ovh/venom/context/default"
	"github.com/ovh/venom/context/webctx"
	"github.com/ovh/venom/executors/dbfixtures"
	"github.com/ovh/venom/executors/exec"
	"github.com/ovh/venom/executors/http"
	"github.com/ovh/venom/executors/imap"
	"github.com/ovh/venom/executors/readfile"
	"github.com/ovh/venom/executors/smtp"
	"github.com/ovh/venom/executors/ssh"
	"github.com/ovh/venom/executors/web"
)

//P is a map of test parameters
type P map[string]interface{}

//V is a map of Variables
type V map[string]string

//R is a map of Results
type R map[string]interface{}

//T is a superset of testing.T
type T struct {
	*testing.T
	v    *venom.Venom
	ts   *venom.TestSuite
	tc   *venom.TestCase
	Name string
}

//Logger is a superset of the testing Logger compliant with logrus Entry
type Logger struct{ t *testing.T }

// Debugf calls testing.T.Logf
func (l *Logger) Debugf(format string, args ...interface{}) { l.t.Logf("[DEBUG] "+format, args...) }

// Infof calls testing.T.Logf
func (l *Logger) Infof(format string, args ...interface{}) { l.t.Logf("[INFO] "+format, args...) }

// Printf calls testing.T.Logf
func (l *Logger) Printf(format string, args ...interface{}) { l.t.Logf(format, args...) }

// Warnf calls testing.T.Logf
func (l *Logger) Warnf(format string, args ...interface{}) { l.t.Logf("[WARN] "+format, args...) }

// Warningf calls testing.T.Logf
func (l *Logger) Warningf(format string, args ...interface{}) { l.t.Logf("[WARN] "+format, args...) }

// Errorf calls testing.T.Logf
func (l *Logger) Errorf(format string, args ...interface{}) { l.t.Logf("[ERROR] "+format, args...) }

// Fatalf calls testing.T.Logf
func (l *Logger) Fatalf(format string, args ...interface{}) { l.t.Logf("[FATAL] "+format, args...) }

// WithField calls testing.T.Logf
func (l *Logger) WithField(key string, value interface{}) venom.Logger {
	return l
}

//TestCase instanciates a veom testcase
func TestCase(t *testing.T, name string, variables map[string]string) *T {
	v := venom.New()
	v.RegisterExecutor(dbfixtures.Name, dbfixtures.New())
	v.RegisterExecutor(exec.Name, exec.New())
	v.RegisterExecutor(http.Name, http.New())
	v.RegisterExecutor(imap.Name, imap.New())
	v.RegisterExecutor(readfile.Name, readfile.New())
	v.RegisterExecutor(smtp.Name, smtp.New())
	v.RegisterExecutor(ssh.Name, ssh.New())
	v.RegisterExecutor(web.Name, web.New())
	v.RegisterTestCaseContext(defaultctx.Name, defaultctx.New())
	v.RegisterTestCaseContext(webctx.Name, webctx.New())

	vt := &T{
		t,
		v,
		&venom.TestSuite{
			Templater: &venom.Templater{Values: variables},
			Name:      name,
		},
		&venom.TestCase{
			Name: name,
		},
		name,
	}

	return vt
}

//Do executes a veom test steps
func (t *T) Do(teststepParams P) R {
	ts := t.ts
	tc := t.tc
	tcc, errContext := t.v.ContextWrap(tc)
	if errContext != nil {
		t.Error(errContext)
		return nil
	}
	if err := tcc.Init(); err != nil {
		tc.Errors = append(tc.Errors, venom.Failure{Value: err.Error()})
		t.Error(err)
		return nil
	}
	defer tcc.Close()

	step, erra := ts.Templater.ApplyOnStep(venom.TestStep(teststepParams))
	if erra != nil {
		t.Error(erra)
		return nil
	}

	e, err := t.v.WrapExecutor(step, tcc)
	if err != nil {
		t.Error(err)
		return nil
	}

	res := t.v.RunTestStep(tcc, e, ts, tc, step, &Logger{t.T})

	for _, f := range tc.Failures {
		t.Errorf("\r Failure %s", f.Value)
	}

	for _, e := range tc.Errors {
		t.Errorf("\r Error %s", e.Value)
	}

	return R(res)
}

//NewExec returns a venom.P properly initialized for exec Executor
func NewExec(script string) P {
	return P{
		"type":   "exec",
		"script": script,
	}
}

var HTTP = struct {
	Get      func(url, path string) P
	Post     func(url, path string, body []byte) P
	PostJSON func(url, path string, body interface{}) P
	Put      func(url, path string, body []byte) P
	PutJSON  func(url, path string, body interface{}) P
	Delete   func(url, path string) P
}{
	Get: func(url, path string) P {
		return P{
			"type":   "http",
			"method": "GET",
			"url":    url,
			"path":   path,
		}
	},
	Post: func(url, path string, body []byte) P {
		return P{
			"type":   "http",
			"method": "POST",
			"url":    url,
			"path":   path,
			"body":   string(body),
		}
	},
	PostJSON: func(url, path string, body interface{}) P {
		b, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		return P{
			"type":   "http",
			"method": "POST",
			"url":    url,
			"path":   path,
			"body":   string(b),
		}
	},
	Put: func(url, path string, body []byte) P {
		return P{
			"type":   "http",
			"method": "PUT",
			"url":    url,
			"path":   path,
			"body":   string(body),
		}
	},
	PutJSON: func(url, path string, body interface{}) P {
		b, err := json.Marshal(body)
		if err != nil {
			panic(err)
		}
		return P{
			"type":   "http",
			"method": "PUT",
			"url":    url,
			"path":   path,
			"body":   string(b),
		}
	},
	Delete: func(url, path string) P {
		return P{
			"type":   "http",
			"method": "DELETE",
			"url":    url,
			"path":   path,
		}
	},
}

func (p P) WithHeaders(headers http.Headers) P {
	p["headers"] = headers
	return p
}
