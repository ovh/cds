package main

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestVariables(t *testing.T) {
	tests := []struct {
		defaults    map[string]interface{}
		vars        map[string]interface{}
		alters      map[string]VariableAlteration
		expected    map[string]interface{}
		shouldCrash bool
	}{
		//0 crash incorrect text template
		{
			defaults: map[string]interface{}{
				"badTemplate": "[[ if gonnacrash ]][[end]]",
			},
			vars:        map[string]interface{}{},
			alters:      nil,
			expected:    nil,
			shouldCrash: true,
		},

		//1 correctly replace the templated ID var
		{
			defaults: map[string]interface{}{
				"1": "Hello [[ .id ]] !",
			},
			vars:   map[string]interface{}{},
			alters: nil,
			expected: map[string]interface{}{
				"id": "tests",
				"1":  "Hello tests !",
			},
			shouldCrash: false,
		},

		//2 correctly replace the templated var
		{
			defaults: map[string]interface{}{
				"1": "Hello [[ .who ]] !",
			},
			vars: map[string]interface{}{
				"who": "world",
			},
			alters: nil,
			expected: map[string]interface{}{
				"id":  "tests",
				"1":   "Hello world !",
				"who": "world",
			},
			shouldCrash: false,
		},

		//3 crash because try to preprend undef default variable
		{
			defaults: map[string]interface{}{},
			vars: map[string]interface{}{
				"1": "... world !",
			},
			alters:      nil,
			expected:    nil,
			shouldCrash: true,
		},

		//4 crash because try to preprend to a non string variable
		{
			defaults: map[string]interface{}{
				"1": 42,
			},
			vars: map[string]interface{}{
				"1": "... world !",
			},
			alters:      nil,
			expected:    nil,
			shouldCrash: true,
		},

		//5 correctly prepend the variable
		{
			defaults: map[string]interface{}{
				"1": "Hello",
			},
			vars: map[string]interface{}{
				"1": "... world !",
			},
			alters: nil,
			expected: map[string]interface{}{
				"id": "tests",
				"1":  "Hello world !",
			},
			shouldCrash: false,
		},

		//6 correctly replace all the variable
		{
			defaults: map[string]interface{}{
				"1": "Hello",
			},
			vars: map[string]interface{}{
				"1": "Hi there",
			},
			alters: nil,
			expected: map[string]interface{}{
				"id": "tests",
				"1":  "Hi there",
			},
			shouldCrash: false,
		},

		//7 crash on key alteration
		{
			defaults: map[string]interface{}{
				"1": "Hello",
			},
			vars: map[string]interface{}{},
			alters: map[string]VariableAlteration{
				"1": func(interface{}) (interface{}, error) {
					return nil, errors.New("crash for the test")
				},
			},
			expected:    nil,
			shouldCrash: true,
		},

		//8 crash on key alteration
		{
			defaults: map[string]interface{}{
				"1": "Hello",
			},
			vars: map[string]interface{}{},
			alters: map[string]VariableAlteration{
				"1": func(v interface{}) (interface{}, error) {
					return strings.ToUpper(v.(string)), nil
				},
			},
			expected: map[string]interface{}{
				"id": "tests",
				"1":  "HELLO",
			},
			shouldCrash: false,
		},

		//9 crash on key alteration
		{
			defaults: map[string]interface{}{
				"1": "Hello",
			},
			vars: map[string]interface{}{},
			alters: map[string]VariableAlteration{
				"1": func(v interface{}) (interface{}, error) {
					return strings.ToUpper(v.(string)), nil
				},
			},
			expected: map[string]interface{}{
				"id": "tests",
				"1":  "HELLO",
			},
			shouldCrash: false,
		},
	}

	for i, test := range tests {
		app := Applications{
			Default: test.defaults,
			Apps: map[string]map[string]interface{}{
				"tests": test.vars,
			},
			alters: test.alters,
		}

		result, err := app.Variables("tests")
		if (err != nil) != test.shouldCrash {
			t.Fatalf("Test #%d failed : should crash (%t) and got : %s", i, test.shouldCrash, err)
		}

		if !reflect.DeepEqual(result, test.expected) {
			jsonResult, _ := json.Marshal(result)
			jsonExpected, _ := json.Marshal(test.expected)

			t.Fatalf("Test #%d failed :\nExpected :\n%s\nGot:\n%s", i, string(jsonExpected), string(jsonResult))
		}
	}
}
