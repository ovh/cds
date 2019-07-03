package hatchery

import (
	"regexp"
	"strings"
)

var (
	// reSplitParams accepts:
	// 	 TEST
	// 	 TEST=
	// 	 TEST=12
	// 	 TEST='12'
	// 	 TEST="12"
	// It does not allow spaces around '='.
	reSplitParams = regexp.MustCompile(`([a-zA-Z_]\w+)(?:=('(?:\\.|[^'\\]+)*'|"(?:\\.|[^"\\]+)*"|\S*))?`)
	// unescapeBackslash allows to unescape a backslash-escaped string.
	unescapeBackslash = strings.NewReplacer(`\\`, `\`, `\`, ``).Replace
)

// ParseRequirementModel parses a requirement model than returns the
// image name and the environment variables.
//
// Example of input:
//   "postgres:latest env_1=blabla env_2=blabla env_3 env_4='zip'"
func ParseRequirementModel(rm string) (string, map[string]string) {
	var env map[string]string

	tuple := strings.SplitN(rm, " ", 2)
	img := tuple[0]

	if len(tuple) > 1 {
		matches := reSplitParams.FindAllStringSubmatch(tuple[1], -1)
		if matches != nil {
			env = make(map[string]string, len(matches))
			for _, m := range matches {
				name, value := m[1], m[2]
				if len(value) >= 2 && (value[0] == '\'' || value[0] == '"') {
					value = unescapeBackslash(value[1 : len(value)-1])
				}
				env[name] = value
			}
		}
	}

	return img, env
}
