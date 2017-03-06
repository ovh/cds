package defaultctx

import "github.com/runabove/venom"

// Context Type name
const Name = "default"

// New returns a new TestCaseContext
func New() venom.TestCaseContext {
	ctx := &DefaultTestCaseContext{}
	ctx.Name = Name
	return ctx
}

// TestCaseContext represents the context of a testcase
type DefaultTestCaseContext struct {
	venom.CommonTestCaseContext
	datas map[string]interface{}
}

// Init Initialize the context
func (tcc *DefaultTestCaseContext) Init() error {
	return nil
}

// Close the context
func (tcc *DefaultTestCaseContext) Close() error {
	return nil
}
