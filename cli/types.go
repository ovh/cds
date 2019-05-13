package cli

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// FlagType for cli flag.
type FlagType string

// Flags types
const (
	FlagString FlagType = "string"
	FlagBool   FlagType = "bool"
	FlagSlice  FlagType = "slice"
	FlagArray  FlagType = "array"
)

// Flag represents a command flag.
type Flag struct {
	Name      string
	ShortHand string
	Usage     string
	Default   string
	Type      FlagType
	IsValid   func(string) bool
}

// Values represents commands flags and args values accessible with their name
type Values map[string][]string

// GetString returns a string.
func (v *Values) GetString(s string) string {
	r := (*v)[s]
	if len(r) == 0 {
		return ""
	}
	return r[0]
}

// GetInt64 returns a int64.
func (v *Values) GetInt64(s string) (int64, error) {
	ns := v.GetString(s)
	if ns == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(ns, 10, 64)
	if err != nil {
		return -1, fmt.Errorf("%s invalid: not a integer", s)
	}
	return n, nil
}

// GetBool returns a string.
func (v *Values) GetBool(s string) bool {
	r := strings.ToLower(v.GetString(s))
	return r == "true" || r == "yes" || r == "y" || r == "1"
}

// GetStringSlice returns a string slice.
func (v *Values) GetStringSlice(s string) []string {
	if strings.TrimSpace(v.GetString(s)) == "" {
		return nil
	}
	res := strings.Split(v.GetString(s), "||")
	if len(res) == 1 && strings.Contains(res[0], ",") {
		return strings.Split(res[0], ",")
	}
	return res
}

// GetStringArray returns a string array.
func (v *Values) GetStringArray(s string) []string {
	return (*v)[s]
}

// Arg represent a command argument
type Arg struct {
	Name       string
	IsValid    func(string) bool
	Weight     int
	AllowEmpty bool
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
	Name         string
	Ctx          []Arg
	Args         []Arg
	OptionalArgs []Arg
	VariadicArgs Arg
	Short        string
	Long         string
	Example      string
	Flags        []Flag
	Aliases      []string
	Hidden       bool
	PreRun       func(c *Command, args *[]string) error
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
				Type:    FlagString,
			},
			{
				Name:  "verbose",
				Usage: "Display all object fields",
				Type:  FlagBool,
			},
			{
				Name:      "quiet",
				ShortHand: "q",
				Default:   "",
				Usage:     "Only display object's key",
				Type:      FlagBool,
			},
			{
				Name:    "fields",
				Default: "",
				Usage:   "Only display specified object fields. 'empty' will display all fields, 'all' will display all object fields, 'field1,field2' to select multiple fields",
				Type:    FlagString,
			},
		}
	case RunListFunc:
		extraFlags = []Flag{
			{
				Name:    "filter",
				Default: "",
				Usage:   "Filter output based on conditions provided",
				Type:    FlagString,
			},
			{
				Name:    "format",
				Default: "table",
				Usage:   "Output format: table|json|yaml",
				Type:    FlagString,
			},
			{
				Name:      "quiet",
				ShortHand: "q",
				Default:   "",
				Usage:     "Only display object's key",
				Type:      FlagBool,
			},
			{
				Name:    "fields",
				Default: "",
				Usage:   "Only display specified object fields. 'empty' will display all fields, 'all' will display all object fields, 'field1,field2' to select multiple fields",
				Type:    FlagString,
			},
			{
				Name:  "verbose",
				Usage: "Display all object fields",
				Type:  FlagBool,
			},
		}
	case RunDeleteFunc:
		extraFlags = []Flag{
			{
				Name:    "force",
				Default: "false",
				Usage:   "Force delete without confirmation and exit 0 if resource does not exist",
				Type:    FlagBool,
			},
		}
	}
	c.Flags = append(c.Flags, extraFlags...)
}

// CommandWithExtraAliases to add common extra alias
func CommandWithExtraAliases(c *Command, run interface{}) {
	var extraAliases []string
	switch run.(type) {
	case RunListFunc:
		extraAliases = []string{"ls"}
	case RunDeleteFunc:
		extraAliases = []string{"rm", "remove", "del"}
	}
	c.Aliases = append(c.Aliases, extraAliases...)
}

// CommandWithPreRun to add pre run function
func CommandWithPreRun(f func(c *Command, args *[]string) error) func(c *Command, run interface{}) {
	return func(c *Command, run interface{}) {
		c.PreRun = f
	}
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

// RunDeleteFunc is a run function for a command. It returns an error.
type RunDeleteFunc func(Values) error

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
