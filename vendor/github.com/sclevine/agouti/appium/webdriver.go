package appium

import (
	"fmt"

	"github.com/sclevine/agouti"
	"github.com/sclevine/agouti/api/mobile"
)

type WebDriver struct {
	driver *agouti.WebDriver
}

func New(options ...Option) *WebDriver {
	newOptions := config{}.merge(options)
	url := "http://{{.Address}}/wd/hub"
	command := []string{"appium", "-p", "{{.Port}}"}
	agoutiWebDriver := agouti.NewWebDriver(url, command, newOptions.agoutiOptions...)
	return &WebDriver{agoutiWebDriver}
}

func (w *WebDriver) NewDevice(options ...Option) (*Device, error) {
	newOptions := config{}.merge(options)
	page, err := w.driver.NewPage(newOptions.agoutiOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebDriver: %s", err)
	}
	mobileSession := &mobile.Session{page.Session()}

	return newDevice(mobileSession, page), nil
}

func (w *WebDriver) Start() error {
	return w.driver.Start()
}

func (w *WebDriver) Stop() error {
	return w.driver.Stop()
}
