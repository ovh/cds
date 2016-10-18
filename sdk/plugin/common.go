package plugin

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/facebookgo/httpcontrol"
)

//HTTP Constants
const (
	MaxTries            = 5
	RequestTimeout      = time.Minute
	AuthHeader          = "X_AUTH_HEADER"
	RequestedWithValue  = "X-CDS-SDK"
	RequestedWithHeader = "X-Requested-With"
)

var (
	auth   IOptions
	client *http.Client
	//Trace is a debug logger
	Trace *log.Logger
)

//Common is the base plugin struct every plugin should be composed by
type Common struct{}

//SetTrace is for debug
func SetTrace(traceHandle io.Writer) {
	Trace = log.New(traceHandle, "TRACE: ", log.Ldate|log.Ltime)
}

//Init is a common function for all plugins
func (p *Common) Init(o IOptions) string {
	SetTrace(ioutil.Discard)
	auth = o

	if auth.TLSSkipVerify() {
		client = &http.Client{
			Transport: &httpcontrol.Transport{
				RequestTimeout: RequestTimeout,
				MaxTries:       MaxTries,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
	} else {
		client = &http.Client{
			Transport: &httpcontrol.Transport{
				RequestTimeout: RequestTimeout,
				MaxTries:       MaxTries,
			},
		}
	}
	return "plugin: initialized on " + o.GetURL()
}

// request executes an authentificated HTTP request on $path given $method
func request(method string, path string, args []byte) ([]byte, int, error) {
	if auth == nil {
		return []byte{}, 0, errors.New("Auth must be initialized")
	}
	var req *http.Request
	var err error
	if args != nil {
		req, err = http.NewRequest(method, auth.GetURL()+path, bytes.NewReader(args))
		if err != nil {
			return nil, 0, err
		}
	} else {
		req, err = http.NewRequest(method, auth.GetURL()+path, nil)
		if err != nil {
			return nil, 0, err
		}
	}

	basedHash := base64.StdEncoding.EncodeToString([]byte(auth.Hash()))
	req.Header.Set(AuthHeader, basedHash)
	req.Header.Set(RequestedWithHeader, RequestedWithValue)
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	code := resp.StatusCode

	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, code, err
	}

	return body, code, nil
}

//Log a a struct to send log to CDS API
type Log struct {
	ActionID int64  `json:"action_build_id"`
	Step     string `json:"step"`
	Value    string `json:"value"`
}

//SendLog send logs to CDS engine for the current
func SendLog(a IAction, step, format string, i ...interface{}) error {
	if a == nil {
		//If action is nil: do nothing
		return nil
	}
	Trace.Printf(format+"\n", i)

	s := fmt.Sprintf(format, i...)
	l := Log{
		ActionID: a.ID(),
		Step:     step,
		Value:    s,
	}
	logs := []Log{l}
	data, err := json.Marshal(logs)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/build/%d/log", logs[0].ActionID)
	_, _, err = request("POST", path, data)
	if err != nil {
		return err
	}
	return nil
}
