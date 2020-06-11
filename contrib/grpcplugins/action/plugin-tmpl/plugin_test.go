package main

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/grpcplugin/actionplugin"
)

func TestRun(t *testing.T) {
	// replace plugin logger

	tmpdir := os.TempDir()
	content := `My name is {{.name}}, I am {{.age}}!`
	params := `name=toto
age=42`

	tmplfile, err := ioutil.TempFile(tmpdir, "plugintmpl")
	if err != nil {
		t.Fatalf("unexpected error creating temporary template file: %s", err)
	}
	defer os.Remove(tmplfile.Name())
	defer tmplfile.Close()

	_, err = tmplfile.WriteString(content)
	if err != nil {
		t.Fatalf("unexpected error writing test content: %s", err)
	}

	p := &tmplActionPlugin{}
	opts := actionplugin.ActionQuery{
		Options: map[string]string{
			"file":   tmplfile.Name(),
			"params": params,
		},
	}

	res, err := p.Run(context.Background(), &opts)
	defer os.Remove(tmplfile.Name() + ".out")

	if err != nil {
		t.Error("should not return error")
		return
	}
	if res.GetStatus() != sdk.StatusSuccess {
		t.Errorf("unexpected error on Run")
		return
	}

	expectedContent := `My name is toto, I am 42!`
	gotContent, err := ioutil.ReadFile(tmplfile.Name() + ".out")
	if err != nil {
		t.Errorf("unexpected error reading generated content: %s", err)
		return
	}

	if expectedContent != string(gotContent) {
		t.Errorf("expected content %q, got %q", expectedContent, gotContent)
		return
	}
}

func TestParseTemplateParameters(t *testing.T) {
	s := `name=toto
age=42`

	params, err := parseTemplateParameters(s)
	if err != nil {
		t.Fatalf("unexpected error parsing template parameters: %s", err)
	}

	expectedParams := map[string]interface{}{
		"name": "toto",
		"age":  "42",
	}

	if !reflect.DeepEqual(params, expectedParams) {
		t.Fatalf("expected %+v, got %+v", expectedParams, params)
	}
}
