package template

import (
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/facebookgo/httpcontrol"
	"github.com/spf13/cobra"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/plugin"
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

	if !strings.Contains(o.Hash(), ":") {
		return "template: init aborted"
	}
	username := strings.Split(o.Hash(), ":")[0]
	token := strings.Split(o.Hash(), ":")[1]

	sdk.Options(o.GetURL(), username, "", token)

	if o.TLSSkipVerify() {
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

// Main func call by template, display info only
func Main(p Interface) {
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

	var rootCmd = &cobra.Command{}
	rootCmd.AddCommand(cmdInfo)
	rootCmd.Execute()
}

// InfoMarkdown returns string formatted with markdown
func InfoMarkdown(t Interface) string {
	var sp string
	ps := t.Parameters()
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
	for _, v := range ps {
		sp += fmt.Sprintf("* **%s**: %s\n", v.Name, v.Description)
	}

	info := fmt.Sprintf(`
%s

## Parameters

%s

## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/templates/%s/README.md)

`,
		t.Description(),
		sp,
		t.Name())

	return info
}
