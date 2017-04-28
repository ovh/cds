package webctx

import (
	"fmt"

	"github.com/sclevine/agouti"

	"github.com/runabove/venom"
)

// Context Type name
const Name = "web"

// Key of context element in testsuite file
const (
	Width  = "width"
	Height = "height"
	Driver = "driver"
	Args   = "args"
)

// New returns a new TestCaseContext
func New() venom.TestCaseContext {
	ctx := &WebTestCaseContext{}
	ctx.Name = Name
	return ctx
}

// TestCaseContex represents the context of a testcase
type WebTestCaseContext struct {
	venom.CommonTestCaseContext
	wd   *agouti.WebDriver
	Page *agouti.Page
}

// BuildContext build context of type web.
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
			}),
		)
	default:
		tcc.wd = agouti.PhantomJS()
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
		var width, height int
		switch tcc.TestCase.Context[Width].(type) {
		case int:
			width = tcc.TestCase.Context[Width].(int)
		default:
			return fmt.Errorf("%s is not an integer: %s", Width, fmt.Sprintf("%s", tcc.TestCase.Context[Width]))
		}
		switch tcc.TestCase.Context[Height].(type) {
		case int:
			height = tcc.TestCase.Context[Height].(int)
		default:
			return fmt.Errorf("%s is not an integer: %s", Height, fmt.Sprintf("%s", tcc.TestCase.Context[Height]))
		}

		if err := tcc.Page.Size(width, height); err != nil {
			return fmt.Errorf("Cannot resize page: %s", err)
		}
	}
	return nil
}

// Close web driver
func (tcc *WebTestCaseContext) Close() error {
	return tcc.wd.Stop()
}
