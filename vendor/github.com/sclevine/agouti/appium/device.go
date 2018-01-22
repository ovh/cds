package appium

import (
	"fmt"

	"github.com/sclevine/agouti"
	"github.com/sclevine/agouti/api/mobile"
	"github.com/sclevine/agouti/internal/element"
)

type mobileSession interface {
	element.Client
	LaunchApp() error
	CloseApp() error
	InstallApp(appPath string) error
	Reset() error
	PerformTouch(actions []mobile.Action) error
	ReplaceValue(elementID, newValue string) error
}

type Device struct {
	*agouti.Page
	session mobileSession
}

func newDevice(session mobileSession, page *agouti.Page) *Device {
	return &Device{
		Page:    page,
		session: session,
	}
}

// Device methods

func (d *Device) LaunchApp() error {
	if err := d.session.LaunchApp(); err != nil {
		return fmt.Errorf("failed to launch app: %s", err)
	}
	return nil
}

func (d *Device) CloseApp() error {
	if err := d.session.CloseApp(); err != nil {
		return fmt.Errorf("failed to close app: %s", err)
	}
	return nil
}

func (d *Device) InstallApp(appPath string) error {
	if err := d.session.InstallApp(appPath); err != nil {
		return fmt.Errorf("failed to install app: %s", err)
	}
	return nil
}

func (d *Device) Reset() error {
	if err := d.session.Reset(); err != nil {
		return fmt.Errorf("failed to reset app: %s", err)
	}
	return nil
}

func (d *Device) TouchAction() *TouchAction {
	return NewTouchAction(d.session)
}

func (d *Device) ReplaceElementValue(element *agouti.Selection, newValue string) error {
	elements, err := element.Elements()
	if err != nil {
		return err
	}

	for _, el := range elements {
		if err := d.session.ReplaceValue(el.GetID(), newValue); err != nil {
			return fmt.Errorf("failed to replace element value: %s", err)
		}
	}

	return nil
}

func (d *Device) Swipe(start_x, start_y, end_x, end_y, duration int) error {
	return d.TouchAction().PressPosition(start_x, start_y).Wait(duration).MoveToPosition(end_x, end_y).Release().Perform()
}
