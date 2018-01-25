package mocks

import "github.com/sclevine/agouti/internal/element"

type ElementRepository struct {
	GetCall struct {
		ReturnElements []element.Element
		Err            error
	}

	GetExactlyOneCall struct {
		ReturnElement element.Element
		Err           error
	}

	GetAtLeastOneCall struct {
		ReturnElements []element.Element
		Err            error
	}
}

func (e *ElementRepository) Get() ([]element.Element, error) {
	return e.GetCall.ReturnElements, e.GetCall.Err
}

func (e *ElementRepository) GetExactlyOne() (element.Element, error) {
	return e.GetExactlyOneCall.ReturnElement, e.GetExactlyOneCall.Err
}

func (e *ElementRepository) GetAtLeastOne() ([]element.Element, error) {
	return e.GetAtLeastOneCall.ReturnElements, e.GetAtLeastOneCall.Err
}
