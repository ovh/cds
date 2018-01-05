package interpolate

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

var interpolateRegex = regexp.MustCompile("({{\\.[a-zA-Z0-9._\\-µ|\\s]+}})")

// Do returns interpolated input with vars
func Do(input string, vars map[string]string) (string, error) {
	if !strings.Contains(input, "{{") {
		return input, nil
	}

	data := make(map[string]string, len(vars))
	defaults := make(map[string]string, len(vars))
	empty := make(map[string]string, len(vars))

	for k, v := range vars {
		kb := strings.Replace(k, ".", "__", -1)
		// "-"" are not a valid char in go template var name
		kb = strings.Replace(kb, "-", "µµµ", -1)
		data[kb] = v

		defaults["."+kb] = k
		if v == "" {
			empty[k] = k
		}

		input = strings.Replace(input, k, kb, -1)
	}

	// in input, replace {{.cds.foo.bar}} with {{ default "{{.cds.foo.bar}}" .cds.foo.bar}}
	// t.Execute will call "default" function, documented on http://masterminds.github.io/sprig/defaults.html
	sm := interpolateRegex.FindAllStringSubmatch(input, -1)
	if len(sm) > 0 {
		alreadyReplaced := map[string]string{}
		for i := 0; i < len(sm); i++ {
			if len(sm[i]) > 0 {
				e := sm[i][1][2 : len(sm[i][1])-2]
				// alreadyReplaced: check if var is already replaced.
				// see test "two same unknown" on DoTest
				if _, ok := alreadyReplaced[sm[i][1]]; !ok {
					if _, ok := defaults[e]; !ok {
						// replace {{.cds.foo.bar}} with {{ default "{{.cds.foo.bar}}" .cds.foo.bar}}
						// with cds.foo.bar unknown from vars
						nameWithDot := strings.Replace(e, "__", ".", -1)
						nameWithDot = strings.Replace(nameWithDot, "µµµ", ".", -1)
						// "-"" are not a valid char in go template var name, as we don't know e, no pb to replace "-" with "µ"
						eb := strings.Replace(e, "-", "µµµ", -1)
						input = strings.Replace(input, sm[i][1], "{{ default \"{{"+nameWithDot+"}}\" "+eb+" }}", -1)
					} else if _, ok := empty[e]; !ok {
						// replace {{.cds.foo.bar}} with {{ default "" .cds.foo.bar}}
						// with cds.foo.bar knowned, but with value empty string ""
						input = strings.Replace(input, sm[i][1], "{{ default \"\" "+e+" }}", -1)
					} else {
						// replace {{.cds.foo.bar}} with {{ default "{{.cds.foo.bar}}" .cds.foo.bar}}
						// with cds.foo.bar knowned from vars
						input = strings.Replace(input, sm[i][1], "{{ default \"{{"+defaults[e[1:]]+"}}\" "+e+" }}", -1)
					}
					alreadyReplaced[sm[i][1]] = sm[i][1]
				}
			}
		}
	}

	t, err := template.New("input").Funcs(interpolateHelperFuncs).Parse(input)
	if err != nil {
		return "", fmt.Errorf("Invalid template format: %s", err.Error())
	}

	var buff bytes.Buffer
	if err := t.Execute(&buff, data); err != nil {
		return "", fmt.Errorf("Failed to execute template: %s", err.Error())
	}

	return buff.String(), nil
}
