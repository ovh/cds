package mocks

import (
	"encoding/json"

	"github.com/sclevine/agouti/api"
)

type Session struct {
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

	GetActiveElementCall struct {
		ReturnElement *api.Element
		Err           error
	}

	DeleteCall struct {
		Called bool
		Err    error
	}

	GetWindowCall struct {
		ReturnWindow *api.Window
		Err          error
	}

	GetWindowsCall struct {
		ReturnWindows []*api.Window
		Err           error
	}

	SetWindowCall struct {
		Window *api.Window
		Err    error
	}

	SetWindowByNameCall struct {
		Name string
		Err  error
	}

	DeleteWindowCall struct {
		Called bool
		Err    error
	}

	GetScreenshotCall struct {
		ReturnImage []byte
		Err         error
	}

	GetCookiesCall struct {
		ReturnCookies []*api.Cookie
		Err           error
	}

	SetCookieCall struct {
		Cookie *api.Cookie
		Err    error
	}

	DeleteCookieCall struct {
		Name string
		Err  error
	}

	DeleteCookiesCall struct {
		Called bool
		Err    error
	}

	GetURLCall struct {
		ReturnURL string
		Err       error
	}

	SetURLCall struct {
		URL string
		Err error
	}

	GetTitleCall struct {
		ReturnTitle string
		Err         error
	}

	GetSourceCall struct {
		ReturnSource string
		Err          error
	}

	MoveToCall struct {
		Element *api.Element
		Offset  api.Offset
		Err     error
	}

	FrameCall struct {
		Frame *api.Element
		Err   error
	}

	FrameParentCall struct {
		Called bool
		Err    error
	}

	ExecuteCall struct {
		Body      string
		Arguments []interface{}
		Result    string
		Err       error
	}

	ForwardCall struct {
		Called bool
		Err    error
	}

	BackCall struct {
		Called bool
		Err    error
	}

	RefreshCall struct {
		Called bool
		Err    error
	}

	GetAlertTextCall struct {
		ReturnText string
		Err        error
	}

	SetAlertTextCall struct {
		Text string
		Err  error
	}

	AcceptAlertCall struct {
		Called bool
		Err    error
	}

	DismissAlertCall struct {
		Called bool
		Err    error
	}

	NewLogsCall struct {
		LogType    string
		ReturnLogs []api.Log
		Err        error
	}

	GetLogTypesCall struct {
		ReturnTypes []string
		Err         error
	}

	DoubleClickCall struct {
		Called bool
		Err    error
	}

	ClickCall struct {
		Button api.Button
		Err    error
	}

	ButtonDownCall struct {
		Button api.Button
		Err    error
	}

	ButtonUpCall struct {
		Button api.Button
		Err    error
	}

	TouchDownCall struct {
		X   int
		Y   int
		Err error
	}

	TouchUpCall struct {
		X   int
		Y   int
		Err error
	}

	TouchMoveCall struct {
		X   int
		Y   int
		Err error
	}

	TouchScrollCall struct {
		Element *api.Element
		Offset  api.Offset
		Err     error
	}

	TouchClickCall struct {
		Element *api.Element
		Err     error
	}

	TouchFlickCall struct {
		Element *api.Element
		Offset  api.Offset
		Speed   api.Speed
		Err     error
	}

	TouchDoubleClickCall struct {
		Element *api.Element
		Err     error
	}

	TouchLongClickCall struct {
		Element *api.Element
		Err     error
	}

	DeleteLocalStorageCall struct {
		Called bool
		Err    error
	}

	DeleteSessionStorageCall struct {
		Called bool
		Err    error
	}

	SetImplicitWaitCall struct {
		Called bool
		Err    error
	}

	SetPageLoadCall struct {
		Called bool
		Err    error
	}

	SetScriptTimeoutCall struct {
		Called bool
		Err    error
	}
}

func (s *Session) Delete() error {
	s.DeleteCall.Called = true
	return s.DeleteCall.Err
}

func (s *Session) GetElement(selector api.Selector) (*api.Element, error) {
	s.GetElementCall.Selector = selector
	return s.GetElementCall.ReturnElement, s.GetElementCall.Err
}

func (s *Session) GetElements(selector api.Selector) ([]*api.Element, error) {
	s.GetElementsCall.Selector = selector
	return s.GetElementsCall.ReturnElements, s.GetElementsCall.Err
}

func (s *Session) GetActiveElement() (*api.Element, error) {
	return s.GetActiveElementCall.ReturnElement, s.GetActiveElementCall.Err
}

func (s *Session) GetWindow() (*api.Window, error) {
	return s.GetWindowCall.ReturnWindow, s.GetWindowCall.Err
}

func (s *Session) GetWindows() ([]*api.Window, error) {
	return s.GetWindowsCall.ReturnWindows, s.GetWindowsCall.Err
}

func (s *Session) SetWindow(window *api.Window) error {
	s.SetWindowCall.Window = window
	return s.SetWindowCall.Err
}

func (s *Session) SetWindowByName(name string) error {
	s.SetWindowByNameCall.Name = name
	return s.SetWindowByNameCall.Err
}

func (s *Session) DeleteWindow() error {
	s.DeleteWindowCall.Called = true
	return s.DeleteWindowCall.Err
}

func (s *Session) GetScreenshot() ([]byte, error) {
	return s.GetScreenshotCall.ReturnImage, s.GetScreenshotCall.Err
}

func (s *Session) GetCookies() ([]*api.Cookie, error) {
	return s.GetCookiesCall.ReturnCookies, s.GetCookiesCall.Err
}

func (s *Session) SetCookie(cookie *api.Cookie) error {
	s.SetCookieCall.Cookie = cookie
	return s.SetCookieCall.Err
}

func (s *Session) DeleteCookie(name string) error {
	s.DeleteCookieCall.Name = name
	return s.DeleteCookieCall.Err
}

func (s *Session) DeleteCookies() error {
	s.DeleteCookiesCall.Called = true
	return s.DeleteCookiesCall.Err
}

func (s *Session) GetURL() (string, error) {
	return s.GetURLCall.ReturnURL, s.GetURLCall.Err
}

func (s *Session) SetURL(url string) error {
	s.SetURLCall.URL = url
	return s.SetURLCall.Err
}

func (s *Session) GetTitle() (string, error) {
	return s.GetTitleCall.ReturnTitle, s.GetTitleCall.Err
}

func (s *Session) GetSource() (string, error) {
	return s.GetSourceCall.ReturnSource, s.GetSourceCall.Err
}

func (s *Session) MoveTo(element *api.Element, offset api.Offset) error {
	s.MoveToCall.Element = element
	s.MoveToCall.Offset = offset
	return s.MoveToCall.Err
}

func (s *Session) Frame(frame *api.Element) error {
	s.FrameCall.Frame = frame
	return s.FrameCall.Err
}

func (s *Session) FrameParent() error {
	s.FrameParentCall.Called = true
	return s.FrameParentCall.Err
}

func (s *Session) Execute(body string, arguments []interface{}, result interface{}) error {
	s.ExecuteCall.Body = body
	s.ExecuteCall.Arguments = arguments
	json.Unmarshal([]byte(s.ExecuteCall.Result), result)
	return s.ExecuteCall.Err
}

func (s *Session) Forward() error {
	s.ForwardCall.Called = true
	return s.ForwardCall.Err
}

func (s *Session) Back() error {
	s.BackCall.Called = true
	return s.BackCall.Err
}

func (s *Session) Refresh() error {
	s.RefreshCall.Called = true
	return s.RefreshCall.Err
}

func (s *Session) GetAlertText() (string, error) {
	return s.GetAlertTextCall.ReturnText, s.GetAlertTextCall.Err
}

func (s *Session) SetAlertText(text string) error {
	s.SetAlertTextCall.Text = text
	return s.SetAlertTextCall.Err
}

func (s *Session) AcceptAlert() error {
	s.AcceptAlertCall.Called = true
	return s.AcceptAlertCall.Err
}

func (s *Session) DismissAlert() error {
	s.DismissAlertCall.Called = true
	return s.DismissAlertCall.Err
}

func (s *Session) NewLogs(logType string) ([]api.Log, error) {
	s.NewLogsCall.LogType = logType
	return s.NewLogsCall.ReturnLogs, s.NewLogsCall.Err
}

func (s *Session) GetLogTypes() ([]string, error) {
	return s.GetLogTypesCall.ReturnTypes, s.GetLogTypesCall.Err
}

func (s *Session) DoubleClick() error {
	s.DoubleClickCall.Called = true
	return s.DoubleClickCall.Err
}

func (s *Session) Click(button api.Button) error {
	s.ClickCall.Button = button
	return s.ClickCall.Err
}

func (s *Session) ButtonDown(button api.Button) error {
	s.ButtonDownCall.Button = button
	return s.ButtonDownCall.Err
}

func (s *Session) ButtonUp(button api.Button) error {
	s.ButtonUpCall.Button = button
	return s.ButtonUpCall.Err
}

func (s *Session) TouchDown(x, y int) error {
	s.TouchDownCall.X = x
	s.TouchDownCall.Y = y
	return s.TouchDownCall.Err
}

func (s *Session) TouchUp(x, y int) error {
	s.TouchUpCall.X = x
	s.TouchUpCall.Y = y
	return s.TouchUpCall.Err
}

func (s *Session) TouchMove(x, y int) error {
	s.TouchMoveCall.X = x
	s.TouchMoveCall.Y = y
	return s.TouchMoveCall.Err
}

func (s *Session) TouchClick(element *api.Element) error {
	s.TouchClickCall.Element = element
	return s.TouchClickCall.Err
}

func (s *Session) TouchDoubleClick(element *api.Element) error {
	s.TouchDoubleClickCall.Element = element
	return s.TouchDoubleClickCall.Err
}

func (s *Session) TouchLongClick(element *api.Element) error {
	s.TouchLongClickCall.Element = element
	return s.TouchLongClickCall.Err
}

func (s *Session) TouchFlick(element *api.Element, offset api.Offset, speed api.Speed) error {
	s.TouchFlickCall.Element = element
	s.TouchFlickCall.Offset = offset
	s.TouchFlickCall.Speed = speed
	return s.TouchFlickCall.Err
}

func (s *Session) TouchScroll(element *api.Element, offset api.Offset) error {
	s.TouchScrollCall.Element = element
	s.TouchScrollCall.Offset = offset
	return s.TouchScrollCall.Err
}

func (s *Session) DeleteLocalStorage() error {
	s.DeleteLocalStorageCall.Called = true
	return s.DeleteLocalStorageCall.Err
}

func (s *Session) DeleteSessionStorage() error {
	s.DeleteSessionStorageCall.Called = true
	return s.DeleteSessionStorageCall.Err
}

func (s *Session) SetImplicitWait(timeout int) error {
	s.SetImplicitWaitCall.Called = true
	return s.SetImplicitWaitCall.Err
}

func (s *Session) SetPageLoad(timeout int) error {
	s.SetPageLoadCall.Called = true
	return s.SetPageLoadCall.Err
}

func (s *Session) SetScriptTimeout(timeout int) error {
	s.SetScriptTimeoutCall.Called = true
	return s.SetScriptTimeoutCall.Err
}
