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
	"sort"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
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
type Common struct {
	Name        string
	Description string
	Parameters  Parameters
	Author      string
	Format      string `json:"-" yaml:"-" xml:"-"`
}

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

// Main func call by plugin, display info only
func Main(p CDSAction) {
	var format string

	var cmdInfo = &cobra.Command{
		Use:   "info",
		Short: "Print plugin Information anything to the screen: info --format <yml>",
		Run: func(cmd *cobra.Command, args []string) {
			if format != "markdown" {
				if err := sdk.Output(format, p, fmt.Printf); err != nil {
					fmt.Printf("Error:%s", err)
				}
				return
			}
			fmt.Print(InfoMarkdown(p))
		},
	}

	cmdInfo.Flags().StringVarP(&format, "format", "", "markdown", "--format:yaml, json, xml, markdown")

	var rootCmd = &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			Serve(p)
		},
	}
	rootCmd.AddCommand(cmdInfo)
	rootCmd.Execute()
}

// InfoMarkdown returns string formatted with markdown
func InfoMarkdown(pl CDSAction) string {
	var sp string
	var keys []string
	for k := range pl.Parameters().DataDescription {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pe := pl.Parameters().DataDescription
	for _, k := range keys {
		v := pe[k]
		sp += fmt.Sprintf("* **%s**: %s\n", k, v)
	}

	info := fmt.Sprintf(`
%s

## Parameters

%s

## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/plugins/%s/README.md)

`,
		pl.Description(),
		sp,
		pl.Name())

	return info
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
	req.Header.Set("User-Agent", "CDS/worker")
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
	PipelineBuildJobID int64                `json:"pipelineBuildJobID"`
	PipelineBuildID    int64                `json:"pipelineBuildID"`
	Start              *timestamp.Timestamp `json:"start"`
	LastModified       *timestamp.Timestamp `json:"lastModified"`
	Done               *timestamp.Timestamp `json:"done"`
	StepOrder          int                  `json:"stepOrder"`
	Value              string               `json:"val"`
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
	if _, _, err := request("POST", path, data); err != nil {
		return err
	}
	return nil
}
