package cli

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// Flag represents a command flag.
type Flag struct {
	Name      string
	ShortHand string
	Usage     string
	Default   string
	Kind      reflect.Kind
	IsValid   func(string) bool
}

// Values represents commands flags and args values accessible with their name
type Values map[string]string

// GetString returns a string
func (v *Values) GetString(s string) string {
	return (*v)[s]
}

// GetBool returns a string
func (v *Values) GetBool(s string) bool {
	return strings.ToLower((*v)[s]) == "true" || strings.ToLower((*v)[s]) == "yes" || strings.ToLower((*v)[s]) == "y" || strings.ToLower((*v)[s]) == "1"
}

// Arg represent a command argument
type Arg struct {
	Name    string
	IsValid func(string) bool
	Weight  int
}

func orderArgs(a ...Arg) args {
	for i := range a {
		if a[i].Weight == 0 {
			a[i].Weight = i
		}

	}
	res := args(a)
	sort.Sort(res)
	return res
}

type args []Arg

// Len is the number of elements in the collection.
func (s args) Len() int {
	return len(s)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s args) Less(i, j int) bool {
	return s[i].Weight < s[j].Weight
}

// Swap swaps the elements with indexes i and j.
func (s args) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Command represents the way to instanciate a cobra.Command
type Command struct {
	Name          string
	Args          []Arg
	OptionnalArgs []Arg
	Short         string
	Long          string
	Flags         []Flag
	Aliases       []string
}

// CommandModifier is a function type to extend a command
type CommandModifier func(*Command, interface{})

// CommandWithoutExtraFlags to avoid add extra flags
func CommandWithoutExtraFlags(c *Command, run interface{}) {}

// CommandWithExtraFlags to add common extra flags
func CommandWithExtraFlags(c *Command, run interface{}) {
	var extraFlags = []Flag{}
	switch run.(type) {
	case RunGetFunc:
		extraFlags = []Flag{
			{
				Name:    "format",
				Default: "plain",
				Usage:   "Output format: plain|json|yaml",
				Kind:    reflect.String,
			},
			{
				Name:  "verbose",
				Usage: "Display all object fields",
				Kind:  reflect.Bool,
			},
		}
	case RunListFunc:
		extraFlags = []Flag{
			{
				Name:    "filter",
				Default: "",
				Usage:   "Filter output based on conditions provided",
				Kind:    reflect.String,
			},
			{
				Name:    "format",
				Default: "table",
				Usage:   "Output format: table|json|yaml",
				Kind:    reflect.String,
			},
			{
				Name:      "quiet",
				ShortHand: "q",
				Default:   "",
				Usage:     "Only display object's key",
				Kind:      reflect.Bool,
			},
			{
				Name:    "fields",
				Default: "",
				Usage:   "Only display specified object fields. 'empty' will display all fields, 'all' will display all object fields, 'field1,field2' to select multiple fields",
				Kind:    reflect.String,
			},
			{
				Name:  "verbose",
				Usage: "Display all object fields",
				Kind:  reflect.Bool,
			},
		}
	}
	c.Flags = append(c.Flags, extraFlags...)
}

// ErrWrongUsage is a common error
var ErrWrongUsage = &Error{1, fmt.Errorf("Wrong usage")}

// Error implements error
type Error struct {
	Code int
	Err  error
}

// Error implements error
func (e *Error) Error() string {
	return e.Err.Error()
}

// ListResult is the result type for command function which returns list. Use AsListResult to compute this
type ListResult []interface{}

// RunFunc is the most basic run function for a command. It returns only an error
type RunFunc func(Values) error

// RunGetFunc is a run function for a command. It returns an object value (not a pointer) and an error.
type RunGetFunc func(Values) (interface{}, error)

// RunListFunc is a run function for a command. It returns an objects list  and an error
type RunListFunc func(Values) (ListResult, error)

// AsListResult compute any slice to ListResult
func AsListResult(i interface{}) ListResult {
	s := reflect.ValueOf(i)
	if s.Kind() != reflect.Slice {
		panic("AsListResult() given a non-slice type")
	}

	res := ListResult{}
	for i := 0; i < s.Len(); i++ {
		v := s.Index(i).Interface()
		res = append(res, v)
	}

	return res
}
