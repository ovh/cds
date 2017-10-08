package sdk

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"

	"github.com/Masterminds/sprig"
)

var interpolateRegex = regexp.MustCompile("{{(\\.[a-zA-Z0-9._|\\s]+)}}")

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
	sm := interpolateRegex.FindAllStringSubmatch(input, -1)
	if len(sm) > 0 {
		alreadyReplaced := map[string]string{}
		for i := 0; i < len(sm); i++ {
			if len(sm[i]) > 0 {
				// alreadyReplaced: check if var is already replaced.
				// see test "two same unknown" on InterpolateTest
				if _, ok := alreadyReplaced[sm[i][1]]; !ok {
					input = strings.Replace(input, sm[i][1], " default \"{{"+sm[i][1]+"}}\" "+sm[i][1]+" ", -1)
					alreadyReplaced[sm[i][1]] = sm[i][1]
				}
			}
		}
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
