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
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
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

// Log struct holds a single line of build log
type Log struct {
	ID                 int64                `json:"id"`
	PipelineBuildJobID int64                `json:"pipeline_build_job_id"`
	PipelineBuildID    int64                `json:"pipeline_build_id"`
	Start              *timestamp.Timestamp `json:"start"`
	LastModified       *timestamp.Timestamp `json:"last_modified"`
	Done               *timestamp.Timestamp `json:"done"`
	StepOrder          int                  `json:"step_order"`
	Value              string               `json:"value"`
}

//SendLog send logs to CDS engine for the current
func SendLog(j IJob, format string, i ...interface{}) error {
	if j == nil {
		//If action is nil: do nothing
		return nil
	}
	Trace.Printf(format+"\n", i)

	now, _ := ptypes.TimestampProto(time.Now())

	s := fmt.Sprintf(format, i...)
	l := Log{
		PipelineBuildJobID: j.ID(),
		PipelineBuildID:    j.PipelineBuildID(),
		Start:              now,
		StepOrder:          j.StepOrder(),
		Value:              s,
		LastModified:       now,
		Done:               &timestamp.Timestamp{},
	}

	data, err := json.Marshal(l)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("/build/%d/log", j.ID())
	_, _, err = request("POST", path, data)
	if err != nil {
		return err
	}
	return nil
}
