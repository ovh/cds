package webctx

import (
	"fmt"
	"time"

	"github.com/sclevine/agouti"

	"github.com/ovh/venom"
)

// Name of the Context
const Name = "web"

// Key of context element in testsuite file
const (
	Width   = "width"
	Height  = "height"
	Driver  = "driver"
	Args    = "args"
	Timeout = "timeout"
	Debug   = "debug"
)

// New returns a new TestCaseContext
func New() venom.TestCaseContext {
	ctx := &WebTestCaseContext{}
	ctx.Name = Name
	return ctx
}

// WebTestCaseContext represents the context of a testcase
type WebTestCaseContext struct {
	venom.CommonTestCaseContext
	wd   *agouti.WebDriver
	Page *agouti.Page
}

// Init build context of type web.
// It creates a new browser
func (tcc *WebTestCaseContext) Init() error {
	var driver string
	if _, ok := tcc.TestCase.Context[Driver]; !ok {
		driver = "phantomjs"
	} else {
		driver = tcc.TestCase.Context[Driver].(string)
	}

	args := []string{}
	if _, ok := tcc.TestCase.Context[Args]; ok {
		switch tcc.TestCase.Context[Args].(type) {
		case []interface{}:
			for _, v := range tcc.TestCase.Context[Args].([]interface{}) {
				args = append(args, v.(string))
			}
		}
	}

	switch driver {
	case "chrome":
		tcc.wd = agouti.ChromeDriver(agouti.Desired(
			agouti.Capabilities{
				"chromeOptions": map[string][]string{
					"args": args,
				},
			}))
	default:
		tcc.wd = agouti.PhantomJS()
	}

	timeout, existTimeout, errTimeout := isIntInContext(tcc.TestCase, Timeout)
	if errTimeout != nil {
		return errTimeout
	}
	if existTimeout {
		tcc.wd.Timeout = time.Duration(timeout) * time.Second
	} else {
		tcc.wd.Timeout = 180 * time.Second // default value
	}
	if v, exist := tcc.TestCase.Context[Debug]; exist {
		switch tcc.TestCase.Context[Debug].(type) {
		case bool:
			tcc.wd.Debug = v.(bool)
		default:
			return fmt.Errorf("%s is not an boolean: %s", Debug, fmt.Sprintf("%s", tcc.TestCase.Context[Debug]))
		}
	}

	if err := tcc.wd.Start(); err != nil {
		return fmt.Errorf("Cannot start web driver %s", err)
	}

	// Init Page
	var errP error
	tcc.Page, errP = tcc.wd.NewPage()
	if errP != nil {
		return fmt.Errorf("Cannot create new page %s", errP)
	}

	resizePage := false
	if _, ok := tcc.TestCase.Context[Width]; ok {
		if _, ok := tcc.TestCase.Context[Height]; ok {
			resizePage = true
		}
	}

	// Resize Page
	if resizePage {
		width, _, errWidth := isIntInContext(tcc.TestCase, Width)
		if errWidth != nil {
			return errWidth
		}
		height, _, errHeight := isIntInContext(tcc.TestCase, Height)
		if errHeight != nil {
			return errHeight
		}
		if err := tcc.Page.Size(width, height); err != nil {
			return fmt.Errorf("Cannot resize page: %s", err)
		}
	}
	return nil
}

// isIntInContext returns  valueOfKey, existOrNot in Context, Error
func isIntInContext(t venom.TestCase, n string) (int, bool, error) {
	if _, exist := t.Context[n]; !exist {
		return -1, false, nil
	}
	switch t.Context[n].(type) {
	case int:
		return t.Context[n].(int), true, nil
	default:
		return -1, true, fmt.Errorf("%s is not an integer: %s", n, fmt.Sprintf("%s", t.Context[n]))
	}
}

// Close web driver
func (tcc *WebTestCaseContext) Close() error {
	return tcc.wd.Stop()
}
