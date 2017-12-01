package ovhapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/go-ovh/ovh"

	"github.com/ovh/venom"
	defaultctx "github.com/ovh/venom/context/default"
	"github.com/ovh/venom/executors"
)

// Name of executor
const Name = "ovhapi"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Headers represents header HTTP for Request
type Headers map[string]string

// Executor struct. Json and yaml descriptor are used for json output
type Executor struct {
	Method   string `json:"method" yaml:"method"`
	NoAuth   bool   `json:"no_auth" yaml:"noAuth"`
	Path     string `json:"path" yaml:"path"`
	Body     string `json:"body" yaml:"body"`
	BodyFile string `json:"bodyfile" yaml:"bodyfile"`
}

// Result represents a step result. Json and yaml descriptor are used for json output
type Result struct {
	Executor    Executor    `json:"executor,omitempty" yaml:"executor,omitempty"`
	TimeSeconds float64     `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string      `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
	StatusCode  int         `json:"statuscode,omitempty" yaml:"statuscode,omitempty"`
	Body        string      `json:"body,omitempty" yaml:"body,omitempty"`
	BodyJSON    interface{} `json:"bodyjson,omitempty" yaml:"bodyjson,omitempty"`
	Err         string      `json:"err,omitempty" yaml:"err,omitempty"`
}

// ZeroValueResult return an empty implemtation of this executor result
func (Executor) ZeroValueResult() venom.ExecutorResult {
	r, _ := executors.Dump(Result{})
	return r
}

// GetDefaultAssertions return default assertions for this executor
// Optional
func (Executor) GetDefaultAssertions() venom.StepAssertions {
	return venom.StepAssertions{Assertions: []string{"result.statuscode ShouldEqual 200"}}
}

// Run execute TestStep
func (Executor) Run(testCaseContext venom.TestCaseContext, l venom.Logger, step venom.TestStep) (venom.ExecutorResult, error) {
	// Get context
	ctx, ok := testCaseContext.(*defaultctx.DefaultTestCaseContext)
	if !ok {
		return nil, fmt.Errorf("ovhapi executor need a default context")
	}

	// transform step to Executor Instance
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}

	// Get context
	var endpoint, applicationKey, applicationSecret, consumerKey string
	var err error
	if endpoint, err = ctx.GetString("endpoint"); err != nil {
		return nil, err
	}
	if !t.NoAuth {
		if applicationKey, err = ctx.GetString("applicationKey"); err != nil {
			return nil, err
		}
		if applicationSecret, err = ctx.GetString("applicationSecret"); err != nil {
			return nil, err
		}
		if consumerKey, err = ctx.GetString("consumerKey"); err != nil {
			return nil, err
		}
	}

	// set default values
	if t.Method == "" {
		t.Method = "GET"
	}

	// init result
	r := Result{Executor: t}

	start := time.Now()
	// prepare ovh api client
	client, err := ovh.NewClient(
		endpoint,
		applicationKey,
		applicationSecret,
		consumerKey,
	)
	if err != nil {
		return nil, err
	}

	// get request body from file or from field
	requestBody, err := t.getRequestBody()
	if err != nil {
		return nil, err
	}

	// do api call
	resp := new(interface{})

	if err = client.CallAPI(t.Method, t.Path, requestBody, resp, !t.NoAuth); err != nil {
		apiError, ok := err.(*ovh.APIError)
		if !ok {
			return nil, err
		}
		r.StatusCode = apiError.Code
		r.Err = apiError.Message
	} else {
		r.StatusCode = 200
	}

	elapsed := time.Since(start)
	r.TimeSeconds = elapsed.Seconds()
	r.TimeHuman = fmt.Sprintf("%s", elapsed)

	// Add response to result body
	if resp != nil {
		r.BodyJSON = *resp
		bb, err := json.Marshal(resp)
		if err != nil {
			return nil, err
		}
		r.Body = string(bb)
	}

	return executors.Dump(r)
}

func (e Executor) getRequestBody() (res interface{}, err error) {
	var bytes []byte
	if e.Body != "" {
		bytes = []byte(e.Body)
	} else if e.BodyFile != "" {
		path := string(e.BodyFile)
		if _, err = os.Stat(path); !os.IsNotExist(err) {
			bytes, err = ioutil.ReadFile(path)
			if err != nil {
				return nil, err
			}
		}
	}
	if len(bytes) > 0 {
		res = new(interface{})
		err = json.Unmarshal(bytes, res)
		return
	}
	return nil, nil
}
