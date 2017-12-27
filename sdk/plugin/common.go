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
	"runtime"
	"sort"
	"strings"
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

//VERSION is set with -ldflags "-X main.VERSION={{.cds.proj.version}}+{{.cds.version}}"
var VERSION = "snapshot"

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

//Version is a common function for all plugins
func (*Common) Version() string {
	return VERSION
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

	var cmdVersion = &cobra.Command{
		Use:   "version",
		Short: "Print plugin version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("CDS Plugin version:%s os:%s architecture:%s\n", sdk.VERSION, runtime.GOOS, runtime.GOARCH)
		},
	}

	cmdInfo.Flags().StringVarP(&format, "format", "", "markdown", "--format:yaml, json, xml, markdown")

	var rootCmd = &cobra.Command{
		Run: func(cmd *cobra.Command, args []string) {
			Serve(p)
		},
	}
	rootCmd.AddCommand(cmdInfo)
	rootCmd.AddCommand(cmdVersion)
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
	BuildID            int64                `json:"builID"`
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

	s := fmt.Sprintf(format, i...)

	for k, v := range j.Secrets().Data {
		if len(v) >= 6 {
			s = strings.Replace(s, v, "**"+k+"**", -1)
		}
	}

	Trace.Println(s)

	now, _ := ptypes.TimestampProto(time.Now())
	l := Log{
		BuildID:            j.ID(),
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

	req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/log", j.WorkerHTTPPort()), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("send log to worker /log: %s", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot send log to worker /log: %s", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("cannot send log to worker /log: HTTP %d", resp.StatusCode)
	}

	return nil
}
