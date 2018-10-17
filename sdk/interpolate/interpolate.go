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
		//The key should be used as a variable to be replaced
		//kAsVar := "{{." + k
		//kbAsVar := "{{." + kb
		//input = strings.Replace(input, kAsVar, kbAsVar, -1)
		//kAsVar = "{{. " + k
		//kbAsVar = "{{. " + kb
		//input = strings.Replace(input, kAsVar, kbAsVar, -1)

		input = strings.Replace(input, "."+k, "."+kb, -1)

	}

	//var helper string
	// in input, replace {{.cds.foo.bar}} with {{ defaultCDS "{{.cds.foo.bar}}" .cds.foo.bar}}
	// t.Execute will call "default" function, documented on http://masterminds.github.io/sprig/defaults.html
	sm := interpolateRegex.FindAllStringSubmatch(input, -1)
	if len(sm) > 0 {
		//		alreadyReplaced := map[string]string{}
		for i := 0; i < len(sm); i++ {
			//			helper = ""
			if len(sm[i]) > 0 {
				fmt.Println("----", sm[i][1], "----", data)

				var expression = strings.TrimSpace(sm[i][1])
				var usedVariables = map[string]struct{}{}
				var usedHelpers = map[string]struct{}{}
				var quotedStuff = []string{}
				var trimmedExpression = strings.TrimPrefix(expression, "{{")
				trimmedExpression = strings.TrimSuffix(trimmedExpression, "}}")
				splittedExpression := strings.Split(trimmedExpression, " ")

				// case: {{"conf"|uvault}}
				if len(splittedExpression) == 1 {
					splittedExpression = strings.Split(trimmedExpression, "|")
				}

				for i, s := range splittedExpression {
					splittedExpression[i] = strings.TrimSpace(s)
					if splittedExpression[i] == "" {
						continue
					}

					switch splittedExpression[i][0] {
					case '.':
						usedVariables[splittedExpression[i][1:]] = struct{}{}
						//usedVariables[strings.Replace(splittedExpression[i][1:], "__", ".", -1)] = struct{}{}
					case '"':
						q := strings.TrimPrefix(splittedExpression[i], "\"")
						q = strings.TrimSuffix(q, "\"")
						quotedStuff = append(quotedStuff, q)
					case '|':
					default:
						usedHelpers[splittedExpression[i][0:]] = struct{}{}
					}
				}

				unknownVariables := []string{}
				for v := range usedVariables {
					if _, is := data[v]; !is {
						unknownVariables = append(unknownVariables, v)
					}
				}

				unknownHelpers := []string{}
				for h := range usedHelpers {
					if _, is := interpolateHelperFuncs[h]; !is {
						unknownHelpers = append(unknownHelpers, h)
					}
				}

				fmt.Println("expression", expression)
				fmt.Println("splittedExpression:", splittedExpression)
				fmt.Println("usedVariables:", usedVariables)
				fmt.Println("usedHelpers:", usedHelpers)
				fmt.Println("unknownVariables:", unknownVariables)
				fmt.Println("unknownHelpers:", unknownHelpers)
				fmt.Println("quotedStuff:", quotedStuff)

				var defaultIsUsed bool
				if _, ok := usedHelpers["default"]; ok {
					defaultIsUsed = true
				}

				if !defaultIsUsed && (len(unknownVariables) > 0 || len(unknownHelpers) > 0) {
					for _, s := range quotedStuff {
						q := strings.Replace(sm[i][1], `"`+s+`"`, `\"`+s+`\"`, -1)
						input = strings.Replace(input, sm[i][1], q, 1)
						sm[i][1] = q
					}

					input = strings.Replace(input, sm[i][1], "{{\""+sm[i][1]+"\"}}", -1)
				}

				/*e := sm[i][1][2 : len(sm[i][1])-2]
				// alreadyReplaced: check if var is already replaced.
				// see test "two same unknown" on DoTest
				if _, ok := alreadyReplaced[sm[i][1]]; !ok {
					if _, ok := defaults[e]; !ok {
						// replace {{.cds.foo.bar}} with {{ defaultCDS "{{.cds.foo.bar}}" .cds.foo.bar}}
						// with cds.foo.bar unknown from vars
						nameWithDot := strings.Replace(e, "__", ".", -1)
						nameWithDot = strings.Replace(nameWithDot, "µµµ", ".", -1)
						nameWithDot = strings.Replace(nameWithDot, "\"", "\\\"", -1)

						helperPos := strings.Index(e, "|")
						// "-"" are not a valid char in go template var name, as we don't know e, no pb to replace "-" with "µ"
						var eb string
						if helperPos > 0 {
							eb = strings.Replace(e[:helperPos], "-", "µµµ", -1) + e[helperPos:]
						} else {
							eb = strings.Replace(e, "-", "µµµ", -1)
						}

						// check if helper exists. if helper does not exist, as
						// '{{"conf"|uvault}}' -> return '{{"conf"|uvault}}' in defaultCDS value
						// '{{ defaultCDS "{{\"conf\"|uvault}}" "" }}'

						if helperPos > 0 && len(eb) > helperPos {
							helper = strings.TrimSpace(eb[helperPos+1:])
							if strings.HasPrefix(helper, "default") {
								// 7 = len("default") --> helper[7:]
								for a, b := range defaults {
									a = strings.TrimPrefix(a, ".")
									eb2 := strings.Replace(eb, b, a, -1)
									smi := strings.Replace(sm[i][1], eb, eb2, 1)
									input = strings.Replace(input, sm[i][1], smi, 1)
								}

							} else if _, ok := interpolateHelperFuncs[helper]; !ok {
								// if the helper is not found we must rewrite the whole thing
								eb = ""
							}
						}

						if !strings.HasPrefix(helper, "default") {
							var s = nameWithDot

							if strings.Contains(helper, "|") {
								helper = strings.TrimSpace(strings.Split(helper, "|")[0])
							}

							var v = strings.TrimSpace(strings.TrimPrefix(strings.Split(s, "|")[0], "."))
							var varExist bool
							_, varExist = vars[v]

							fmt.Println("v exist ?", v, varExist)

							if _, ok := interpolateHelperFuncs[helper]; ok && strings.Contains(s, "|") && varExist {
								s = "." + v
							}

							fmt.Println("eb:", eb)
							input = strings.Replace(input, sm[i][1], "{{ defaultCDS \"{{"+s+"}}\" "+eb+" }}", -1)
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
					fmt.Println("input:", sm[i][1])
					alreadyReplaced[sm[i][1]] = sm[i][1]
				}*/
			}
		}
	}

	fmt.Println("input:", input)

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
