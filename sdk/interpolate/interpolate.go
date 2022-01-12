package interpolate

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"text/template"
)

var interpolateRegex = regexp.MustCompile("({{[\\.\"a-zA-Z0-9._\\-µ|\\s]+}})")

type void struct{}
type val map[string]interface{}

func (v val) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		_, _ = io.WriteString(s, fmt.Sprintf("%v", v["_"]))
	}
}

// Do returns interpolated input with vars
func Do(input string, vars map[string]string) (string, error) {
	if !strings.Contains(input, "{{") {
		return input, nil
	}

	data := make(val, len(vars))
	flatData := make(map[string]string, len(vars))

	// sort key, to replace the longer variables before
	keys := make([]string, len(vars))
	var i int64
	for k := range vars {
		keys[i] = k
		i++
	}
	sort.Slice(keys, func(i, j int) bool {
		return strings.Count(keys[i], ".") > strings.Count(keys[j], ".")
	})

	replacements := make([]string, 0, 1000)

	for _, k := range keys {
		// handle "-" in var name
		kb := strings.Replace(k, "-", "µµµ", -1)

		//Split the keys by dot
		tokens := strings.Split(kb, ".")
		tmp := &data
		for i := 0; i < len(tokens)-1; i++ {
			_, exist := (*tmp)[tokens[i]]
			if !exist {
				(*tmp)[tokens[i]] = &val{}
			}
			tmp = (*tmp)[tokens[i]].(*val)
		}

		// This is useful to manage {{.cds.env.lb.prefix}}.{{.cds.env.lb}}
		if existingVal, has := (*tmp)[tokens[len(tokens)-1]]; has {
			x, ok := existingVal.(*val)
			if ok {
				(*x)["_"] = vars[k]
			}
			(*tmp)[tokens[len(tokens)-1]] = x
		} else {
			(*tmp)[tokens[len(tokens)-1]] = vars[k]
		}

		// this is used to check the variables later
		flatData[kb] = vars[k]

		// handle "-" in var name
		replacements = append(replacements, "."+k+" ", "."+kb+" ", "."+k+"}", "."+kb+"}", "."+k+"|", "."+kb+"|")
	}

	// handle "-" in var name
	replacer := strings.NewReplacer(replacements...)
	input = replacer.Replace(input)

	var processedExpression = map[string]void{}
	sm := interpolateRegex.FindAllStringSubmatch(input, -1)
	if len(sm) > 0 {
		var usedVariables = make(map[string]void, len(vars))
		var usedHelpers = make(map[string]void, len(InterpolateHelperFuncs))
		for i := 0; i < len(sm); i++ {
			if len(sm[i]) > 0 {
				var expression = strings.TrimSpace(sm[i][1])
				if _, ok := processedExpression[expression]; ok {
					continue
				}
				processedExpression[expression] = void{}

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
						usedVariables[splittedExpression[i][1:]] = void{}
					case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
						quotedStuff = append(quotedStuff, splittedExpression[i:]...)
					case '"':
						q := strings.TrimPrefix(splittedExpression[i], "\"")
						q = strings.TrimSuffix(q, "\"")
						quotedStuff = append(quotedStuff, q)
					case '|':
					default:
						usedHelpers[splittedExpression[i][0:]] = void{}
					}
				}

				var isHandlingUnknownVars bool
				if _, ok := usedHelpers["default"]; ok {
					isHandlingUnknownVars = true
				}
				if _, ok := usedHelpers["ternary"]; ok {
					isHandlingUnknownVars = true
				}

				unknownVariables := make([]string, 0, 1000)
				for v := range usedVariables {
					if _, is := flatData[v]; !is {
						unknownVariables = append(unknownVariables, v)
					}
					delete(usedVariables, v)
				}

				unknownHelpers := make([]string, 0, 1000)
				for h := range usedHelpers {
					if _, is := InterpolateHelperFuncs[h]; !is {
						unknownHelpers = append(unknownHelpers, h)
					}
					delete(usedHelpers, h)
				}

				if !isHandlingUnknownVars && (len(unknownVariables) > 0 || len(unknownHelpers) > 0) {
					for _, s := range quotedStuff {
						q := strings.Replace(sm[i][1], `"`+s+`"`, `\"`+s+`\"`, -1)
						input = strings.Replace(input, sm[i][1], q, 1)
						sm[i][1] = q
					}

					input = strings.Replace(input, sm[i][1], "{{\""+sm[i][1]+"\"}}", -1)
				}
			}
		}
	}

	t, err := template.New("input").Funcs(InterpolateHelperFuncs).Parse(input)
	if err != nil {
		return "", fmt.Errorf("invalid template format \"%s\": %s", input, err.Error())
	}

	var buff bytes.Buffer
	if err := t.Execute(&buff, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %s", err.Error())
	}

	return buff.String(), nil
}
