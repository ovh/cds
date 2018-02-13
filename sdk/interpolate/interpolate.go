package interpolate

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var interpolateRegex = regexp.MustCompile("({{[\\.\"a-zA-Z0-9._\\-µ|\\s]+}})")

type reverseString []string

func (p reverseString) Len() int           { return len(p) }
func (p reverseString) Less(i, j int) bool { return p[i] > p[j] }
func (p reverseString) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

// Do returns interpolated input with vars
func Do(input string, vars map[string]string) (string, error) {
	if !strings.Contains(input, "{{") {
		return input, nil
	}

	data := make(map[string]string, len(vars))
	defaults := make(map[string]string, len(vars))
	empty := make(map[string]string, len(vars))

	// sort key, to replace the longer variables before
	// see "same prefix" unit test
	keys := make([]string, len(vars))
	var i int64
	for k := range vars {
		keys[i] = k
		i++
	}
	sort.Sort(reverseString(keys))

	for _, k := range keys {
		v := vars[k]
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

	var helper string
	// in input, replace {{.cds.foo.bar}} with {{ defaultCDS "{{.cds.foo.bar}}" .cds.foo.bar}}
	// t.Execute will call "default" function, documented on http://masterminds.github.io/sprig/defaults.html
	sm := interpolateRegex.FindAllStringSubmatch(input, -1)
	if len(sm) > 0 {
		alreadyReplaced := map[string]string{}
		for i := 0; i < len(sm); i++ {
			helper = ""
			if len(sm[i]) > 0 {
				e := sm[i][1][2 : len(sm[i][1])-2]
				// alreadyReplaced: check if var is already replaced.
				// see test "two same unknown" on DoTest
				if _, ok := alreadyReplaced[sm[i][1]]; !ok {
					if _, ok := defaults[e]; !ok {
						// replace {{.cds.foo.bar}} with {{ defaultCDS "{{.cds.foo.bar}}" .cds.foo.bar}}
						// with cds.foo.bar unknown from vars
						nameWithDot := strings.Replace(e, "__", ".", -1)
						nameWithDot = strings.Replace(nameWithDot, "µµµ", ".", -1)
						nameWithDot = strings.Replace(nameWithDot, "\"", "\\\"", -1)
						// "-"" are not a valid char in go template var name, as we don't know e, no pb to replace "-" with "µ"
						eb := strings.Replace(e, "-", "µµµ", -1)

						// check if helper exists. if helper does not exist, as
						// '{{"conf"|uvault}}' -> return '{{"conf"|uvault}}' in defaultCDS value
						// '{{ defaultCDS "{{\"conf\"|uvault}}" "" }}'

						if pos := strings.Index(eb, "|"); pos > 0 && len(eb) > pos {
							helper = strings.TrimSpace(eb[pos+1:])
							if strings.HasPrefix(helper, "default") {
								// 7 = len("default") --> helper[7:]
								input = strings.Replace(input, sm[i][1], "{{ defaultCDS "+helper[7:]+" "+eb+" }}", -1)
							} else if _, ok := interpolateHelperFuncs[helper]; !ok {
								eb = ""
							}
						}
						if helper != "default" {
							input = strings.Replace(input, sm[i][1], "{{ defaultCDS \"{{"+nameWithDot+"}}\" "+eb+" }}", -1)
						}
					} else if _, ok := empty[e]; !ok {
						// replace {{.cds.foo.bar}} with {{ defaultCDS "" .cds.foo.bar}}
						// with cds.foo.bar knowned, but with value empty string ""
						input = strings.Replace(input, sm[i][1], "{{ defaultCDS \"\" "+e+" }}", -1)
					} else {
						// replace {{.cds.foo.bar}} with {{ defaultCDS "{{.cds.foo.bar}}" .cds.foo.bar}}
						// with cds.foo.bar knowned from vars
						input = strings.Replace(input, sm[i][1], "{{ defaultCDS \"{{"+defaults[e[1:]]+"}}\" "+e+" }}", -1)
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
