package dump

import (
	"bytes"
	"io"
	"os"
)

// Dump displays the passed parameter properties to standard out such as complete types and all
// pointer addresses used to indirect to the final value.
// See Fdump if you would prefer dumping to an arbitrary io.Writer or Sdump to
// get the formatted result as a string.
func Dump(i interface{}, formatters ...KeyFormatterFunc) error {
	return Fdump(os.Stdout, i, formatters...)
}

// Sdump returns a string with the passed arguments formatted exactly the same as Dump.
func Sdump(i interface{}, formatters ...KeyFormatterFunc) (string, error) {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	w := new(bytes.Buffer)
	e := NewDefaultEncoder(w)
	e.Formatters = formatters
	return e.Sdump(i)
}

// Fdump formats and displays the passed arguments to io.Writer w. It formats exactly the same as Dump.
func Fdump(w io.Writer, i interface{}, formatters ...KeyFormatterFunc) error {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	e := NewDefaultEncoder(w)
	e.Formatters = formatters
	return e.Fdump(i)
}

// ToMap dumps argument as a map[string]interface{}
func ToMap(i interface{}, formatters ...KeyFormatterFunc) (map[string]interface{}, error) {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	w := new(bytes.Buffer)
	e := NewDefaultEncoder(w)
	e.Formatters = formatters
	return e.ToMap(i)
}

// ToStringMap formats the argument as a map[string]string. It formats exactly the same as Dump.
func ToStringMap(i interface{}, formatters ...KeyFormatterFunc) (map[string]string, error) {
	if formatters == nil {
		formatters = []KeyFormatterFunc{WithDefaultFormatter()}
	}
	w := new(bytes.Buffer)
	e := NewDefaultEncoder(w)
	e.Formatters = formatters
	return e.ToStringMap(i)
}

// MustSdump is a helper that wraps a call to a function returning (string, error)
// and panics if the error is non-nil.
func MustSdump(i interface{}, formatters ...KeyFormatterFunc) string {
	enc := NewDefaultEncoder(new(bytes.Buffer))
	s, err := enc.Sdump(i)
	if err != nil {
		panic(err)
	}
	return s
}
