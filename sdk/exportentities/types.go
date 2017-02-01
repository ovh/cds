package exportentities

import (
	"errors"
	"text/template"
)

type (
	Format int

	HCLable interface {
		HCLTemplate() (*template.Template, error)
	}

	// VariableValue is a struct to export a value of Variable
	VariableValue struct {
		Type  string `json:"type" yaml:"type"`
		Value string `json:"value" yaml:"value"`
	}
)

//All the consts
const (
	FormatJSON Format = iota
	FormatYAML
	FormatHCL
	UnknownFormat
)

var (
	UnsupportedHCLFormat = errors.New("HCL Format is not supported for this entity")
	UnsupportedFormat    = errors.New("Format is not supported")
)
