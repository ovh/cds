package hatchery

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"
)

var (
	// reSplitParams accepts:
	// 	 TEST
	// 	 TEST=
	// 	 TEST=12
	// 	 TEST='12'
	// 	 TEST='1\'2'
	// 	 TEST="12"
	// 	 TEST="1\"2"
	// It does not allow spaces around '='.
	splitParams = regexp.MustCompile(`([a-zA-Z_]\w+)(?:=('(?:\\.|[^'\\]+)*'|"(?:\\.|[^"\\]+)*"|\S*))?`).FindAllStringSubmatch

	// unescapeBackslash allows to unescape a backslash-escaped string.
	unescapeBackslash = strings.NewReplacer(`\\`, `\`, `\`, ``).Replace
)

func quoted(s string) bool {
	if len(s) >= 2 {
		switch s[0] {
		case '\'':
			return s[len(s)-1] == '\''
		case '"':
			return s[len(s)-1] == '"'
		}
	}
	return false
}

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
		matches := splitParams(tuple[1], -1)
		if matches != nil {
			env = make(map[string]string, len(matches))
			for _, m := range matches {
				name, value := m[1], m[2]
				if quoted(value) {
					value = unescapeBackslash(value[1 : len(value)-1])
				}
				// non-quoted values cannot be escaped here
				env[name] = value
			}
		}
	}

	return img, env
}

// ParseArgs splits str on spaces into a slice of strings taking into
// account any quoting (using '' or "") even inside args, and any
// backslash-escaping even without quotes:
//   `abc   def`       → ["abc", "def"]
//   ` abc def `       → ["abc", "def"]
//   ` '' "" `         → ["", ""]
//   ` a'bc' d"e"f `   → ["abc", "def"]
//   `'a\bc\'' "def" ` → ["abc'", "def"]
//   ` abc\ def `      → ["abc def"]
func ParseArgs(str string) []string {
	str = strings.TrimSpace(str)
	if str == "" {
		return nil
	}

	var (
		quoted rune
		cur    bytes.Buffer
		bs, sp bool
	)

	var ret []string
	for _, r := range str {
		if sp {
			if unicode.IsSpace(r) {
				continue
			}
			sp = false
		}

		if bs {
			cur.WriteRune(r)
			bs = false
			continue
		}

		if r == '\\' {
			bs = true
			continue
		}

		// currently quoted
		if quoted != 0 {
			if r == quoted { // close quoting
				quoted = 0
			} else {
				cur.WriteRune(r)
			}
			continue
		}

		switch r {
		case '"', '\'': // open quoting
			quoted = r
			continue
		}

		if unicode.IsSpace(r) {
			ret = append(ret, cur.String())
			cur.Truncate(0)
			sp = true
			continue
		}

		cur.WriteRune(r)
	}

	// Last backslash is let as is
	if bs {
		cur.WriteRune('\\')
	}
	return append(ret, cur.String())
}
