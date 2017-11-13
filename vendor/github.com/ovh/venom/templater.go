package venom

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v2"
)

// Templater contains templating values on a testsuite
type Templater struct {
	Values map[string]string
}

func newTemplater(inputValues map[string]string) *Templater {
	// Copy map to be thread safe with parallel > 1
	values := make(map[string]string)
	for key, value := range inputValues {
		values[key] = value
	}
	return &Templater{Values: values}
}

// Add add data to templater
func (tmpl *Templater) Add(prefix string, values map[string]string) {
	if tmpl.Values == nil {
		tmpl.Values = make(map[string]string)
	}
	dot := ""
	if prefix != "" {
		dot = "."
	}
	for k, v := range values {
		tmpl.Values[prefix+dot+k] = v
	}
}

//ApplyOnStep executes the template on a test step
func (tmpl *Templater) ApplyOnStep(step TestStep) (TestStep, error) {
	// Using yaml to encode/decode, it generates map[interface{}]interface{} typed data that json does not like
	s, err := yaml.Marshal(step)
	if err != nil {
		return nil, fmt.Errorf("templater> Error while marshaling: %s", err)
	}
	sb := tmpl.apply(s)

	var t TestStep
	if err := yaml.Unmarshal([]byte(sb), &t); err != nil {
		return nil, fmt.Errorf("templater> Error while unmarshal: %s, content:%s", err, sb)
	}

	return t, nil
}

//ApplyOnContext executes the template on a context
func (tmpl *Templater) ApplyOnContext(ctx map[string]interface{}) (map[string]interface{}, error) {
	// Using yaml to encode/decode, it generates map[interface{}]interface{} typed data that json does not like
	s, err := yaml.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("templater> Error while marshaling: %s", err)
	}
	sb := tmpl.apply(s)

	var t map[string]interface{}
	if err := yaml.Unmarshal([]byte(sb), &t); err != nil {
		return nil, fmt.Errorf("templater> Error while unmarshal: %s, content:%s", err, sb)
	}

	return t, nil
}

func (tmpl *Templater) apply(in []byte) []byte {
	// Apply template values on values themselves first.
	tmpValues := make(map[string]string)
	for k1, v1 := range tmpl.Values {
		for k2, v2 := range tmpl.Values {
			v1 = strings.Replace(v1, "{{."+k2+"}}", v2, -1)
		}
		tmpValues[k1] = v1
	}
	out := string(in)
	for k, v := range tmpValues {
		out = strings.Replace(out, "{{."+k+"}}", v, -1)
	}
	return []byte(out)
}
