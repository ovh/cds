package exportentities

import (
	"errors"
)

type (
	//Format is a type
	Format int

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
	FormatTOML
	UnknownFormat
)

var (
	// ErrUnsupportedFormat is for unknown format
	ErrUnsupportedFormat = errors.New("Format is not supported")
)
