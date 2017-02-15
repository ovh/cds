package venom

import (
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/ovh/cds/sdk"
)

// Test represents a TestStep. See plugins for implementation
type Test interface {
	// Check checks result
	Check(*sdk.TestCase, *sdk.TestStep, string, *log.Entry)
	// Run run a Test Case
	Run(*sdk.TestStep, *log.Entry, map[string]string)
	// GetDefaultAssertion returns default assertion
	GetDefaultAssertion(string) string
}

var (
	testTypesLock sync.Mutex
	testTypes     = map[string]func() Test{}
)

// RegisterTestFactory register Test Plugins
func RegisterTestFactory(name string, factory func() Test) {
	testTypesLock.Lock()
	testTypes[name] = factory
	testTypesLock.Unlock()
}

// newTest initializes a test by name
func newTest(name string) Test {
	testTypesLock.Lock()
	f, ok := testTypes[name]
	testTypesLock.Unlock()
	if !ok {
		return nil
	}
	return f()
}
