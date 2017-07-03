package cli

import (
	"fmt"
	"reflect"
	"sort"
)

type Flag struct {
	Name      string
	ShortHand string
	Usage     string
	Default   string
	IsValid   func(string) bool
}

type Values map[string]string

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

type Command struct {
	Name          string
	Args          []Arg
	OptionnalArgs []Arg
	Short         string
	Long          string
	Flags         []Flag
}

type CommandModifier func(*Command, interface{})

func CommandWithoutExtraFlags(c *Command, run interface{}) {}

func CommandWithExtraFlags(c *Command, run interface{}) {
	var extraFlags = []Flag{}
	switch run.(type) {
	case RunGetFunc:
		extraFlags = []Flag{
			{
				Name:    "format",
				Default: "plain",
				Usage:   "Output format: plain|json|yaml",
			},
			{
				Name:    "quiet",
				Default: "",
				Usage:   "Only display object's key",
			},
		}
	case RunListFunc:
		extraFlags = []Flag{
			{
				Name:    "filter",
				Default: "",
				Usage:   "Filter output based on conditions provided",
			},
			{
				Name:    "format",
				Default: "plain",
				Usage:   "Output format: table|json|yaml",
			},
			{
				Name:      "quiet",
				ShortHand: "q",
				Default:   "",
				Usage:     "Only display object's key",
			},
			{
				Name:    "fields",
				Default: "",
				Usage:   "Only display specified object fields. 'empty' will display common fields, 'all' will display all object fields, 'field1,field2' to select multiple fields",
			},
		}
	}
	c.Flags = append(c.Flags, extraFlags...)
}

var ErrWrongUsage = &Error{1, fmt.Errorf("Wrong usage")}

type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string {
	return e.Err.Error()
}

type GetResult interface{}
type ListResult []interface{}

type RunFunc func(Values) error
type RunGetFunc func(Values) (GetResult, error)
type RunListFunc func(Values) (ListResult, error)

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
