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

func newTemplater(values map[string]string) *Templater {
	if values == nil {
		values = make(map[string]string)
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

func (tmpl *Templater) applyOnStep(step TestStep) (TestStep, error) {
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

func (tmpl *Templater) apply(in []byte) []byte {
	out := string(in)
	for k, v := range tmpl.Values {
		out = strings.Replace(out, "{{."+k+"}}", v, -1)
	}
	return []byte(out)
}
