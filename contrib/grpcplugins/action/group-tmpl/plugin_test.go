package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"text/template"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func TestRun(t *testing.T) {
	outputfile := "result.json"

	tmpdir := os.TempDir()
	config := `
    {
        "id": "{{.id}}",
        "var": "{{.var}}",
        "prepend": "{{.prepend}}"
    }`

	applications := `
    {
        "default": {
            "var": "toto",
            "prepend": "Hello"
        },
        "apps": {
            "first": {
                "prepend": "... world !"
            },
            "second": {
                "var": "titi"
            }
        }
    }`

	expectedContent := `
    {
         "apps": [
             {
                "id": "first",
                "var": "toto",
                "prepend": "Hello world !"
             },
             {
                "id": "second",
                "var": "titi",
                "prepend": "Hello"
             }
        ]
    }`

	configfile, err := ioutil.TempFile(tmpdir, "config.tmpl")
	if err != nil {
		t.Fatalf("unexpected error creating temporary config file: %s", err)
	}
	defer os.Remove(configfile.Name())
	defer configfile.Close()

	_, err = configfile.WriteString(config)
	if err != nil {
		t.Fatalf("unexpected error writing config content: %s", err)
	}

	applicationsfile, err := ioutil.TempFile(tmpdir, "applications.json")
	if err != nil {
		t.Fatalf("unexpected error creating temporary applications file: %s", err)
	}
	defer os.Remove(applicationsfile.Name())
	defer applicationsfile.Close()

	_, err = applicationsfile.WriteString(applications)
	if err != nil {
		t.Fatalf("unexpected error writing applications content: %s", err)
	}

	action := &actionplugin.ActionQuery{
		Options: map[string]string{
			"config":       configfile.Name(),
			"applications": applicationsfile.Name(),
			"id":           "/group-test",
			"output":       outputfile,
		},
		JobID: 42,
	}

	p := &groupTmplActionPlugin{}

	res, err := p.Run(context.Background(), action)
	if err != nil || res == nil {
		t.Errorf("Unexpected error on run %v", err)
		return
	}
	defer os.Remove(outputfile)

	if res.GetStatus() != sdk.StatusSuccess {
		t.Errorf("unexpected error on Run: %s", res.GetDetails())
		return
	}

	gotContent, err := ioutil.ReadFile(outputfile)
	if err != nil {
		t.Fatalf("unexpected error reading generated content: %s", err)
		return
	}

	var got, expected map[string]interface{}
	if err := json.Unmarshal(gotContent, &got); err != nil {
		t.Fatalf("unexpected error unmarshal generated content: %s", err)
	}

	if err := json.Unmarshal([]byte(expectedContent), &expected); err != nil {
		t.Fatalf("unexpected error unmarshal generated content: %s", err)
	}

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("expected content %s, got %s", expectedContent, gotContent)
		return
	}
}

func TestGetConfigByApplication(t *testing.T) {
	tests := []struct {
		apps        *Applications
		tmpl        *template.Template
		expected    map[string]string
		shouldCrash bool
	}{
		//0 Basic
		{
			apps: &Applications{
				Apps: map[string]map[string]interface{}{
					"test": {
						"a": 1,
						"b": 2,
						"c": 3,
					},
				},
			},
			tmpl: template.Must(template.New("header").Parse(`a:{{.a}};b:{{.b}};c:{{.c}}`)),
			expected: map[string]string{
				"test": `a:1;b:2;c:3`,
			},
		},

		//1 with defaults
		{
			apps: &Applications{
				Default: map[string]interface{}{
					"b": 2,
					"c": 43,
				},
				Apps: map[string]map[string]interface{}{
					"test": {
						"a": 1,
						"c": 3,
					},
				},
			},
			tmpl: template.Must(template.New("header").Parse(`a:{{.a}};b:{{.b}};c:{{.c}}`)),
			expected: map[string]string{
				"test": `a:1;b:2;c:3`,
			},
		},
	}

	for i, test := range tests {
		result, err := getConfigByApplication(test.apps, test.tmpl)

		if (err != nil) != test.shouldCrash {
			t.Fatalf("Test #%d failed : it should crash %t and got %s", i, test.shouldCrash, err)
		}

		if !reflect.DeepEqual(result, test.expected) {
			t.Fatalf("Test #%d failed : expected %+v, got %+v", i, test.expected, result)
		}
	}
}
