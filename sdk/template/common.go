package template

import (
	"crypto/tls"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/ovh/cds/sdk/plugin"

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
	auth   plugin.IOptions
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
func (p *Common) Init(o plugin.IOptions) string {
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

	return "template: initialized on " + o.GetURL()
}
