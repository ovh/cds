package http

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/fsamin/go-dump"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/engine/venom"
)

// Name of executor
const Name = "http"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Headers represents header HTTP for Request
type Headers map[string]string

// Executor struct
type Executor struct {
	Method  string  `json:"method" yaml:"method"`
	URL     string  `json:"url" yaml:"url"`
	Path    string  `json:"path" yaml:"path"`
	Body    string  `json:"body" yaml:"body"`
	Headers Headers `json:"headers" yaml:"headers"`
}

// Result represents a step result
type Result struct {
	Executor    Executor `json:"executor,omitempty" yaml:"executor,omitempty"`
	TimeSeconds float64  `json:"timeSeconds,omitempty" yaml:"timeSeconds,omitempty"`
	TimeHuman   string   `json:"timeHuman,omitempty" yaml:"timeHuman,omitempty"`
	StatusCode  int      `json:"statusCode,omitempty" yaml:"statusCode,omitempty"`
	Body        string   `json:"body,omitempty" yaml:"body,omitempty"`
	Headers     Headers  `json:"headers,omitempty" yaml:"headers,omitempty"`
	Err         error    `json:"error,omitempty" yaml:"error,omitempty"`
}

// GetDefaultAssertions return default assertions for this executor
// Optional
func (Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{"result.code ShouldEqual 0"}}
}

// Run execute TestStep
func (Executor) Run(l *log.Entry, aliases venom.Aliases, step venom.TestStep) (venom.ExecutorResult, error) {

	// transform step to Executor Instance
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}

	r := Result{Executor: t}
	var body io.Reader

	path := t.URL + t.Path
	method := t.Method
	if method == "" {
		method = "GET"
	}
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()
	r.TimeHuman = fmt.Sprintf("%s", elapsed)

	var bb []byte
	if resp.Body != nil {
		defer resp.Body.Close()
		var errr error
		bb, errr = ioutil.ReadAll(resp.Body)
		if errr != nil {
			return nil, errr
		}
		r.Body = string(bb)
	}

	r.Headers = make(map[string]string)

	for k, v := range resp.Header {
		r.Headers[k] = v[0]
	}

	r.StatusCode = resp.StatusCode

	return dump.ToMap(r, dump.WithDefaultLowerCaseFormatter())
}
