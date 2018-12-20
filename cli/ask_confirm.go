package cli

import (
	"fmt"
	"log"
	"strings"

	"github.com/AlecAivazis/survey"
)

// AskForConfirmation ask for yes/no confirmation on command line.
func AskForConfirmation(s string) bool {
	var result bool

	if err := survey.AskOne(&survey.Confirm{
		Message: s,
		Default: true,
	}, &result, nil); err != nil {
		log.Fatal(err)
	}

	return result
}

// MultiChoice for multiple choices question. It returns the selected option
func MultiChoice(s string, opts ...string) int {
	var result string

	if err := survey.AskOne(&survey.Select{
		Message:  s,
		Options:  opts,
		PageSize: 10,
	}, &result, nil); err != nil {
		log.Fatal(err)
	}

	for i := range opts {
		if opts[i] == result {
			return i
		}
	}

	return 0
}

// AskValueChoice ask for a string and returns it.
func AskValueChoice(s string) string {
	var result string

	if err := survey.AskOne(&survey.Input{
		Message: s,
	}, &result, nil); err != nil {
		log.Fatal(err)
	}

	return strings.TrimSpace(result)
}

// CustomMultiSelect is a custom multi select over survey multi select
// that allows to add extra info on items.
type CustomMultiSelect struct {
	survey.MultiSelect
	optionsMap map[string]CustomMultiSelectOption
	Message    string
	Options    []CustomMultiSelectOption
}

// Init survey multi select from options.
func (c *CustomMultiSelect) Init() {
	c.optionsMap = make(map[string]CustomMultiSelectOption)

	allOptions := make([]string, len(c.Options))
	var defaultOptions []string
	for i := range c.Options {
		allOptions[i] = fmt.Sprintf("%s (%s)", c.Options[i].Value, c.Options[i].Info)
		c.optionsMap[allOptions[i]] = c.Options[i]
		if c.Options[i].Default {
			defaultOptions = append(defaultOptions, allOptions[i])
		}
	}
	c.MultiSelect = survey.MultiSelect{
		Message: c.Message,
		Options: allOptions,
		Default: defaultOptions,
	}
}

// Prompt override to extract option values.
func (c *CustomMultiSelect) Prompt() (interface{}, error) {
	resMultiSelect, err := c.MultiSelect.Prompt()
	if err != nil {
		return nil, err
	}

	resMultiSelectStrings := resMultiSelect.([]string)
	results := make([]string, len(resMultiSelectStrings))
	for i := range resMultiSelectStrings {
		results[i] = c.optionsMap[resMultiSelectStrings[i]].Value
	}

	return results, nil
}

// CustomMultiSelectOption for CustomMultiSelect.
type CustomMultiSelectOption struct {
	Value   string
	Info    string
	Default bool
}
