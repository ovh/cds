package integration_test

import (
	"os"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sclevine/agouti"
)

var (
	phantomDriver    = agouti.PhantomJS()
	chromeDriver     = agouti.ChromeDriver()
	seleniumDriver   = agouti.Selenium(agouti.Browser("firefox"))
	selendroidDriver = agouti.Selendroid("selendroid-standalone-0.15.0-with-dependencies.jar")
	edgeDriver       = agouti.EdgeDriver()

	headlessOnly = os.Getenv("HEADLESS_ONLY") == "true"
	mobile       = os.Getenv("MOBILE") == "true"
	windowsOnly  = runtime.GOOS == "windows"
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	Expect(phantomDriver.Start()).To(Succeed())
	if !headlessOnly {
		Expect(chromeDriver.Start()).To(Succeed())
		Expect(seleniumDriver.Start()).To(Succeed())
	}

	if windowsOnly {
		Expect(edgeDriver.Start()).To(Succeed())
	}
	if mobile {
		Expect(selendroidDriver.Start()).To(Succeed())
	}
})

var _ = AfterSuite(func() {
	Expect(phantomDriver.Stop()).To(Succeed())
	if !headlessOnly {
		Expect(chromeDriver.Stop()).To(Succeed())
		Expect(seleniumDriver.Stop()).To(Succeed())
	}
	if windowsOnly {
		Expect(edgeDriver.Stop()).To(Succeed())
	}
	if mobile {
		Expect(selendroidDriver.Stop()).To(Succeed())
	}
})
