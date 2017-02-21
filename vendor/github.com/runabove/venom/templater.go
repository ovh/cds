package venom

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Templater contains templating values on a testsuite
type Templater struct {
	Values map[string]string
}

func newTemplater(values map[string]string) *Templater {
	return &Templater{Values: values}
}

// Add add data to templater
func (tmpl *Templater) Add(prefix string, values map[string]string) {
	for k, v := range values {
		tmpl.Values[prefix+"."+k] = v
	}
}

// Apply apply vars on string
func (tmpl *Templater) Apply(step TestStep) (TestStep, error) {

	log.Debugf("templater> before: %+v", step)

	s, err := json.Marshal(step)
	if err != nil {
		return nil, fmt.Errorf("templater> Error while marshaling: %s", err)
	}
	sb := string(s)

	for k, v := range tmpl.Values {
		sb = strings.Replace(sb, "{{."+k+"}}", v, -1)
	}

	var t TestStep
	if err := json.Unmarshal([]byte(sb), &t); err != nil {
		return nil, fmt.Errorf("templater> Error while unmarshal: %s", err)
	}

	log.Debugf("templater> after: %+v", t)

	return t, nil
}
