package sdk

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/Masterminds/sprig"
)

// Interpolate returns interpolated input with vars
func Interpolate(input string, vars map[string]string) (string, error) {
	data := map[string]string{}

	for k, v := range vars {
		kb := strings.Replace(k, ".", "__", -1)
		data[kb] = v
		re := regexp.MustCompile("{{." + k + "(.*)}}")
		for i := 0; i < 10; i++ {
			sm := re.FindStringSubmatch(input)
			if len(sm) > 0 {
				input = strings.Replace(input, sm[0], "{{."+kb+sm[1]+"}}", -1)
			} else {
				break
			}
		}
	}

	// in input, replace {{.cds.foo.bar}} per {{ default "{{.cds.foo.bar}}" .cds.foo.bar}}
	// t.Execute will call "default" function, documented on http://masterminds.github.io/sprig/defaults.html
	re := regexp.MustCompile("{{\\.(.*)}}")
	sm := re.FindStringSubmatch(input)
	if len(sm) > 0 {
		input = strings.Replace(input, sm[0], "{{ default \"{{."+sm[1]+"}}\" ."+sm[1]+"}}", -1)
	}

	t, err := template.New("input").Funcs(sprig.FuncMap()).Parse(input)
	if err != nil {
		return "", fmt.Errorf("Invalid template format: %s", err.Error())
	}

	var buff bytes.Buffer
	if err := t.Execute(&buff, data); err != nil {
		return "", fmt.Errorf("Failed to execute template: %s", err.Error())
	}

	return buff.String(), nil
}
