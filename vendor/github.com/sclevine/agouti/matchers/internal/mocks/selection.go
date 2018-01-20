package mocks

type Selection struct {
	StringCall struct {
		ReturnString string
	}

	TextCall struct {
		ReturnText string
		Err        error
	}

	AttributeCall struct {
		Attribute   string
		ReturnValue string
		Err         error
	}

	CSSCall struct {
		Property    string
		ReturnValue string
		Err         error
	}

	SelectedCall struct {
		ReturnSelected bool
		Err            error
	}

	VisibleCall struct {
		ReturnVisible bool
		Err           error
	}

	EnabledCall struct {
		ReturnEnabled bool
		Err           error
	}

	ActiveCall struct {
		ReturnActive bool
		Err          error
	}

	CountCall struct {
		ReturnCount int
		Err         error
	}

	EqualsElementCall struct {
		Selection    interface{}
		ReturnEquals bool
		Err          error
	}
}

func (s *Selection) String() string {
	return s.StringCall.ReturnString
}

func (s *Selection) Text() (string, error) {
	return s.TextCall.ReturnText, s.TextCall.Err
}

func (s *Selection) Attribute(attribute string) (string, error) {
	s.AttributeCall.Attribute = attribute
	return s.AttributeCall.ReturnValue, s.AttributeCall.Err
}

func (s *Selection) CSS(property string) (string, error) {
	s.CSSCall.Property = property
	return s.CSSCall.ReturnValue, s.CSSCall.Err
}

func (s *Selection) Selected() (bool, error) {
	return s.SelectedCall.ReturnSelected, s.SelectedCall.Err
}

func (s *Selection) Visible() (bool, error) {
	return s.VisibleCall.ReturnVisible, s.VisibleCall.Err
}

func (s *Selection) Enabled() (bool, error) {
	return s.EnabledCall.ReturnEnabled, s.EnabledCall.Err
}

func (s *Selection) Active() (bool, error) {
	return s.ActiveCall.ReturnActive, s.ActiveCall.Err
}

func (s *Selection) Count() (int, error) {
	return s.CountCall.ReturnCount, s.CountCall.Err
}

func (s *Selection) EqualsElement(selection interface{}) (bool, error) {
	s.EqualsElementCall.Selection = selection
	return s.EqualsElementCall.ReturnEquals, s.EqualsElementCall.Err
}
