package sdk

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
)

// InterpolateFilterFunc is the type of a filter func
type InterpolateFilterFunc func() (string, func(string) string)

// InterpolateFilters provides some standers filters
var InterpolateFilters = struct {
	Title  InterpolateFilterFunc
	Lower  InterpolateFilterFunc
	Upper  InterpolateFilterFunc
	Escape InterpolateFilterFunc
}{
	Title: func() (string, func(string) string) {
		return "title", strings.Title
	},
	Lower: func() (string, func(string) string) {
		return "lower", strings.ToLower
	},
	Upper: func() (string, func(string) string) {
		return "upper", strings.ToUpper
	},
	Escape: func() (string, func(string) string) {
		return "escape", func(s string) string {
			s1 := strings.Replace(s, "_", "-", -1)
			s1 = strings.Replace(s1, "/", "-", -1)
			s1 = strings.Replace(s1, ".", "-", -1)
			return s1
		}
	},
}

// Interpolate returns interpolated input with vars
func Interpolate(input string, vars map[string]string, filters ...InterpolateFilterFunc) (string, error) {
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

	funcMap := template.FuncMap{}
	for i := range filters {
		s, fun := filters[i]()
		funcMap[s] = fun
	}

	t, err := template.New("input").Funcs(funcMap).Parse(input)
	if err != nil {
		return "", fmt.Errorf("Invalid template format: %s", err.Error())
	}

	var buff bytes.Buffer
	if err := t.Execute(&buff, data); err != nil {
		return "", fmt.Errorf("Failed to execute template: %s", err.Error())
	}

	return buff.String(), nil
}
