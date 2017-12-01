package dump

import (
	"reflect"
	"strings"
)

// KeyFormatterFunc is a type for key formatting
type KeyFormatterFunc func(s string) string

// WithLowerCaseFormatter formats keys in lowercase
func WithLowerCaseFormatter() KeyFormatterFunc {
	return func(s string) string {
		return strings.ToLower(s)
	}
}

// WithDefaultLowerCaseFormatter formats keys in lowercase and apply default formatting
func WithDefaultLowerCaseFormatter() KeyFormatterFunc {
	f := WithDefaultFormatter()
	return func(s string) string {
		return strings.ToLower(f(s))
	}
}

// WithDefaultFormatter is the default formatter
func WithDefaultFormatter() KeyFormatterFunc {
	return func(s string) string {
		s = strings.Replace(s, " ", "_", -1)
		s = strings.Replace(s, "/", "_", -1)
		s = strings.Replace(s, ":", "_", -1)
		return s
	}
}

// NoFormatter doesn't do anything, so to be sure to avoid keys formatting, use only this formatter
func NoFormatter() KeyFormatterFunc {
	return func(s string) string {
		return s
	}
}

func valueFromInterface(i interface{}) reflect.Value {
	var f reflect.Value
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		f = reflect.ValueOf(i).Elem()
	} else {
		f = reflect.ValueOf(i)
	}

	if f.Kind() == reflect.Interface {
		if reflect.ValueOf(f.Interface()).Kind() == reflect.Ptr {
			f = reflect.ValueOf(f.Interface()).Elem()
		} else {
			f = reflect.ValueOf(f.Interface())
		}
	}
	return f
}

func validAndNotEmpty(v reflect.Value) bool {
	if v.IsValid() && v.CanInterface() {
		if v.Kind() == reflect.String {
			return v.String() != ""
		}
		return true
	}
	return false
}

func sliceFormat(s []string, formatters []KeyFormatterFunc) []string {
	for i := range s {
		s[i] = format(s[i], formatters)
	}
	return s
}

func format(s string, formatters []KeyFormatterFunc) string {
	for _, f := range formatters {
		s = f(s)
	}
	return s
}
