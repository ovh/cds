package integration_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/sclevine/agouti"
)

var _ = Describe("integration tests", func() {
	testPage("PhantomJS", phantomDriver.NewPage)
	testSelection("PhantomJS", phantomDriver.NewPage)

	if !headlessOnly {
		testPage("ChromeDriver", chromeDriver.NewPage)
		testSelection("ChromeDriver", chromeDriver.NewPage)
		testPage("Firefox", seleniumDriver.NewPage)
		testSelection("Firefox", seleniumDriver.NewPage)
	}
	if windowsOnly {
		testPage("Edge", edgeDriver.NewPage)
		testSelection("Edge", edgeDriver.NewPage)
	}

	if mobile {
		testMobile("Android", selendroidDriver.NewPage)
	}
})

type pageFunc func(...agouti.Option) (*agouti.Page, error)
