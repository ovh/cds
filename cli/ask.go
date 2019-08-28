package cli

import (
	"fmt"
	"log"
	"strings"

	survey "gopkg.in/AlecAivazis/survey.v1"
)

// AskConfirm for confirmation on command line.
func AskConfirm(s string) bool {
	var result bool

	if err := survey.AskOne(&survey.Confirm{
		Message: s,
		Default: true,
	}, &result, nil); err != nil {
		log.Fatal(err)
	}

	return result
}

// AskValue ask for a string and returns it.
func AskValue(s string) string {
	var result string

	if err := survey.AskOne(&survey.Input{
		Message: s,
	}, &result, nil); err != nil {
		log.Fatal(err)
	}

	return strings.TrimSpace(result)
}

// AskChoice for a choice in given options, returns the selected option index.
func AskChoice(s string, opts ...string) int {
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

// AskSelect for multiple choices in given options, returns indexes of selected options.
func AskSelect(s string, opts ...string) []int {
	var results []string

	if err := survey.AskOne(&survey.MultiSelect{
		Message:  s,
		Options:  opts,
		PageSize: 10,
	}, &results, nil); err != nil {
		log.Fatal(err)
	}

	var choices []int
	for i := range opts {
		for j := range results {
			if opts[i] == results[j] {
				choices = append(choices, i)
			}
		}
	}

	return choices
}

// NewCustomMultiSelect custom survey multi select from options.
func NewCustomMultiSelect(message string, opts ...CustomMultiSelectOption) *CustomMultiSelect {
	c := &CustomMultiSelect{
		Message: message,
		Options: opts,
	}

	c.optionsMap = make(map[string]CustomMultiSelectOption, len(c.Options))

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

	return c
}

// CustomMultiSelect is a custom multi select over survey multi select
// that allows to add extra info on items.
type CustomMultiSelect struct {
	survey.MultiSelect
	optionsMap map[string]CustomMultiSelectOption
	Message    string
	Options    []CustomMultiSelectOption
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
