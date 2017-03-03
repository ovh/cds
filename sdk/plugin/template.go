package plugin

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// ApplyArguments apply plugin Arguments on a string
// replace {{.cds.var... }} by values from cds
func ApplyArguments(variables map[string]string, in []byte) ([]byte, error) {
	input := string(in)
	data := map[string]string{}
	for k, v := range variables {
		kb := strings.Replace(k, ".", "__", -1)
		data[kb] = v
		re := regexp.MustCompile("{{." + k + "(.*)}}")
		for {
			sm := re.FindStringSubmatch(input)
			if len(sm) > 0 {
				input = strings.Replace(input, sm[0], "{{."+kb+sm[1]+"}}", -1)
			} else {
				break
			}
		}
	}

	funcMap := template.FuncMap{
		"title":  strings.Title,
		"lower":  strings.ToLower,
		"upper":  strings.ToUpper,
		"escape": Escape,
	}

	t, err := template.New("input").Funcs(funcMap).Parse(input)
	if err != nil {
		return []byte{}, fmt.Errorf("Invalid template format: %s\n", err.Error())
	}

	var buff bytes.Buffer
	if err := t.Execute(&buff, data); err != nil {
		return []byte{}, fmt.Errorf("Failed to execute template: %s\n", err.Error())
	}

	return buff.Bytes(), nil
}

// Escape replace '_', '/', '.' with '-'
func Escape(s string) string {
	s1 := strings.Replace(s, "_", "-", -1)
	s1 = strings.Replace(s1, "/", "-", -1)
	s1 = strings.Replace(s1, ".", "-", -1)
	return s1
}
