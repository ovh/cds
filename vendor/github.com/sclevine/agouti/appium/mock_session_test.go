package appium_test

import (
	"github.com/sclevine/agouti/api"
	"github.com/sclevine/agouti/api/mobile"
)

type mockMobileSession struct {
	GetElementCall struct {
		Selector      api.Selector
		ReturnElement *api.Element
		Err           error
	}

	GetElementsCall struct {
		Selector       api.Selector
		ReturnElements []*api.Element
		Err            error
	}

	PerformTouchCall struct {
		Selector      api.Selector
		ReturnElement *api.Element
		Err           error
	}

	LaunchAppCall struct {
		Err error
	}

	CloseAppCall struct {
		Err error
	}

	InstallAppCall struct {
		Err error
	}

	ResetCall struct {
		Err error
	}

	ReplaceValueCall struct {
		Err error
	}
}

func (s *mockMobileSession) GetElement(selector api.Selector) (*api.Element, error) {
	s.GetElementCall.Selector = selector
	return s.GetElementCall.ReturnElement, s.GetElementCall.Err
}

func (s *mockMobileSession) GetElements(selector api.Selector) ([]*api.Element, error) {
	s.GetElementsCall.Selector = selector
	return s.GetElementsCall.ReturnElements, s.GetElementsCall.Err
}

func (s *mockMobileSession) LaunchApp() error {
	return s.LaunchAppCall.Err
}

func (s *mockMobileSession) CloseApp() error {
	return s.CloseAppCall.Err
}

func (s *mockMobileSession) InstallApp(appPath string) error {
	return s.InstallAppCall.Err
}

func (s *mockMobileSession) Reset() error {
	return s.ResetCall.Err
}

func (s *mockMobileSession) ReplaceValue(elementId, newValue string) error {
	return s.ReplaceValueCall.Err
}

func (s *mockMobileSession) PerformTouch(actions []mobile.Action) error {
	return s.PerformTouchCall.Err
}
