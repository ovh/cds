package http

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/ovh/venom"
	"github.com/ovh/venom/executors"
)

// Name of executor
const Name = "http"

// New returns a new Executor
func New() venom.Executor {
	return &Executor{}
}

// Headers represents header HTTP for Request
type Headers map[string]string

// Executor struct. Json and yaml descriptor are used for json output
type Executor struct {
	Method            string      `json:"method" yaml:"method"`
	URL               string      `json:"url" yaml:"url"`
	Path              string      `json:"path" yaml:"path"`
	Body              string      `json:"body" yaml:"body"`
	BodyFile          string      `json:"bodyfile" yaml:"bodyfile"`
	MultipartForm     interface{} `json:"multipart_form" yaml:"multipart_form"`
	Headers           Headers     `json:"headers" yaml:"headers"`
	IgnoreVerifySSL   bool        `json:"ignore_verify_ssl" yaml:"ignore_verify_ssl" mapstructure:"ignore_verify_ssl"`
	BasicAuthUser     string      `json:"basic_auth_user" yaml:"basic_auth_user" mapstructure:"basic_auth_user"`
	BasicAuthPassword string      `json:"basic_auth_password" yaml:"basic_auth_password" mapstructure:"basic_auth_password"`
}

// Result represents a step result. Json and yaml descriptor are used for json output
type Result struct {
	Executor    Executor    `json:"executor,omitempty" yaml:"executor,omitempty"`
	TimeSeconds float64     `json:"timeseconds,omitempty" yaml:"timeseconds,omitempty"`
	TimeHuman   string      `json:"timehuman,omitempty" yaml:"timehuman,omitempty"`
	StatusCode  int         `json:"statuscode,omitempty" yaml:"statuscode,omitempty"`
	Body        string      `json:"body,omitempty" yaml:"body,omitempty"`
	BodyJSON    interface{} `json:"bodyjson,omitempty" yaml:"bodyjson,omitempty"`
	Headers     Headers     `json:"headers,omitempty" yaml:"headers,omitempty"`
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
	t0 := time.Now()
	l.Debugf("http.Run> Begin")
	defer func() {
		l.Debugf("http.Run> End (%.3f seconds)", time.Since(t0).Seconds())
	}()

	// transform step to Executor Instance
	var t Executor
	if err := mapstructure.Decode(step, &t); err != nil {
		return nil, err
	}

	// dirty: mapstructure doesn't like decoding map[interface{}]interface{}, let's force manually
	t.MultipartForm = step["multipart_form"]

	r := Result{Executor: t}

	req, err := t.getRequest()
	if err != nil {
		return nil, err
	}

	for k, v := range t.Headers {
		req.Header.Set(k, v)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: t.IgnoreVerifySSL},
	}
	client := &http.Client{Transport: tr}

	start := time.Now()
	l.Debugf("http.Run.doRequest> Begin")
	resp, err := client.Do(req)
	l.Debugf("http.Run.doRequest> End (%.3f seconds)", time.Since(t0).Seconds())
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

		bodyJSONArray := []interface{}{}
		if err := json.Unmarshal(bb, &bodyJSONArray); err != nil {
			bodyJSONMap := map[string]interface{}{}
			if err2 := json.Unmarshal(bb, &bodyJSONMap); err2 == nil {
				r.BodyJSON = bodyJSONMap
			}
		} else {
			r.BodyJSON = bodyJSONArray
		}
	}

	r.Headers = make(map[string]string)

	for k, v := range resp.Header {
		r.Headers[k] = v[0]
	}
	r.StatusCode = resp.StatusCode

	return executors.Dump(r)
}

// getRequest returns the request correctly set for the current executor
func (e Executor) getRequest() (*http.Request, error) {
	path := fmt.Sprintf("%s%s", e.URL, e.Path)
	method := e.Method
	if method == "" {
		method = "GET"
	}
	if (e.Body != "" || e.BodyFile != "") && e.MultipartForm != nil {
		return nil, fmt.Errorf("Can only use one of 'body', 'body_file' and 'multipart_form'")
	}
	body := &bytes.Buffer{}
	var writer *multipart.Writer
	if e.Body != "" {
		body = bytes.NewBuffer([]byte(e.Body))
	} else if e.BodyFile != "" {
		path := string(e.BodyFile)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			temp, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, err
			}
			body = bytes.NewBuffer(temp)
		}
	} else if e.MultipartForm != nil {
		form, ok := e.MultipartForm.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("'multipart_form' should be a map")
		}
		writer = multipart.NewWriter(body)
		for k, v := range form {
			key, ok := k.(string)
			if !ok {
				return nil, fmt.Errorf("'multipart_form' should be a map with keys as strings")
			}
			value, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("'multipart_form' should be a map with values as strings")
			}
			// Considering file will be prefixed by @ (since you could also post regular data in the body)
			if strings.HasPrefix(value, "@") {
				// todo: how can we be sure the @ is not the value we wanted to use ?
				if _, err := os.Stat(value[1:]); !os.IsNotExist(err) {
					part, err := writer.CreateFormFile(key, filepath.Base(value[1:]))
					if err != nil {
						return nil, err
					}
					if err := writeFile(part, value[1:]); err != nil {
						return nil, err
					}
					continue
				}
			}
			if err := writer.WriteField(key, value); err != nil {
				return nil, err
			}
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	if len(e.BasicAuthUser) > 0 || len(e.BasicAuthPassword) > 0 {
		req.SetBasicAuth(e.BasicAuthUser, e.BasicAuthPassword)
	}

	if writer != nil {
		req.Header.Set("Content-Type", writer.FormDataContentType())
	}
	return req, err
}

// writeFile writes the content of the file to an io.Writer
func writeFile(part io.Writer, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(part, file)
	return err
}
