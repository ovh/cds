package venom

import (
	"bytes"
	"fmt"
	"strings"
	"time"

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
func (tmpl *Templater) ApplyOnStep(stepNumber int, step TestStep) (TestStep, error) {
	// Using yaml to encode/decode, it generates map[interface{}]interface{} typed data that json does not like
	s, err := yaml.Marshal(step)
	if err != nil {
		return nil, fmt.Errorf("templater> Error while marshaling: %s", err)
	}
	sb := s
	// if the testTest use some variable, we run tmpl.apply on it
	if strings.Contains(string(s), "{{") {
		if stepNumber >= 0 {
			tmpl.Add("", map[string]string{"venom.teststep.number": fmt.Sprintf("%d", stepNumber)})
		}
		sb = tmpl.apply(s)
	}

	var t TestStep
	if err := yaml.Unmarshal([]byte(sb), &t); err != nil {
		return nil, fmt.Errorf("templater> Error while unmarshal: %s, content:%s", err, sb)
	}

	return t, nil
}

//ApplyOnContext executes the template on a context
func (tmpl *Templater) ApplyOnContext(ctx map[string]interface{}) (map[string]interface{}, error) {
	var t map[string]interface{}
	if len(ctx) == 0 {
		return t, nil
	}

	// Using yaml to encode/decode, it generates map[interface{}]interface{} typed data that json does not like
	s, err := yaml.Marshal(ctx)
	if err != nil {
		return nil, fmt.Errorf("templater> Error while marshaling: %s", err)
	}
	sb := s
	// if the context use some variable, we run tmpl.apply on it
	if strings.Contains(string(s), "{{") {
		sb = tmpl.apply(s)
	}

	if err := yaml.Unmarshal([]byte(sb), &t); err != nil {
		return nil, fmt.Errorf("templater> Error while unmarshal: %s, content:%s", err, sb)
	}

	return t, nil
}

func (tmpl *Templater) apply(in []byte) []byte {
	tmpl.Add("", map[string]string{
		"venom.datetime":  time.Now().Format(time.RFC3339),
		"venom.timestamp": fmt.Sprintf("%d", time.Now().Unix()),
	})
	out := string(in)
	for k, v := range tmpl.Values {
		var buffer bytes.Buffer
		buffer.WriteString("{{.")
		buffer.WriteString(k)
		buffer.WriteString("}}")
		out = strings.Replace(out, buffer.String(), v, -1)
		// if no more variable to replace, exit
		if !strings.Contains(out, "{{") {
			return []byte(out)
		}
	}
	return []byte(out)
}
