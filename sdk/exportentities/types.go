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

	// ParameterValue is a struct to export a defautl value of Parameter
	ParameterValue struct {
		Type         string `json:"type" yaml:"type"`
		DefaultValue string `json:"default" yaml:"default"`
	}
)

//All the consts
const (
	FormatJSON Format = iota
	FormatYAML
	FormatHCL
	FormatTOML
	UnknownFormat
)

var (
	// ErrUnsupportedHCLFormat is the error for unsupported HCL format
	ErrUnsupportedHCLFormat = errors.New("HCL Format is not supported for this entity")
	// ErrUnsupportedFormat is for unknown format
	ErrUnsupportedFormat = errors.New("Format is not supported")
)
