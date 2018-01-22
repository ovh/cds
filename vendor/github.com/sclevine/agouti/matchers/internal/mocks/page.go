package mocks

import "github.com/sclevine/agouti"

type Page struct {
	TitleCall struct {
		ReturnTitle string
		Err         error
	}

	PopupTextCall struct {
		ReturnText string
		Err        error
	}

	URLCall struct {
		ReturnURL string
		Err       error
	}

	WindowCountCall struct {
		ReturnCount int
		Err         error
	}

	ReadAllLogsCall struct {
		LogType    string
		ReturnLogs []agouti.Log
		Err        error
	}
}

func (*Page) String() string {
	return "page"
}

func (p *Page) Title() (string, error) {
	return p.TitleCall.ReturnTitle, p.TitleCall.Err
}

func (p *Page) PopupText() (string, error) {
	return p.PopupTextCall.ReturnText, p.PopupTextCall.Err
}

func (p *Page) URL() (string, error) {
	return p.URLCall.ReturnURL, p.URLCall.Err
}

func (p *Page) WindowCount() (int, error) {
	return p.WindowCountCall.ReturnCount, p.WindowCountCall.Err
}

func (p *Page) ReadAllLogs(logType string) ([]agouti.Log, error) {
	p.ReadAllLogsCall.LogType = logType
	return p.ReadAllLogsCall.ReturnLogs, p.ReadAllLogsCall.Err
}
