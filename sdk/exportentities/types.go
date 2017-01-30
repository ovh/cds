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
