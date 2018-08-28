package venom

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/mitchellh/mapstructure"
)

// applyExtracts try to run extract on step, return true if all extracts are OK, false otherwise
func applyExtracts(executorResult *ExecutorResult, step TestStep) assertionsApplied {
	var se StepExtracts
	var errors []Failure
	var failures []Failure

	if err := mapstructure.Decode(step, &se); err != nil {
		return assertionsApplied{
			ok:     false,
			errors: []Failure{{Value: RemoveNotPrintableChar(fmt.Sprintf("error decoding extracts: %s", err))}},
		}
	}

	isOK := true
	for key, pattern := range se.Extracts {
		e := *executorResult
		if _, ok := e[key]; !ok {
			return assertionsApplied{
				ok:     false,
				errors: []Failure{{Value: RemoveNotPrintableChar(fmt.Sprintf("key %s in result is not found", key))}},
			}
		}
		errs, fails := checkExtracts(transformPattern(pattern), fmt.Sprintf("%v", e[key]), executorResult)
		if errs != nil {
			errors = append(errors, *errs)
			isOK = false
		}
		if fails != nil {
			failures = append(failures, *fails)
			isOK = false
		}
	}

	return assertionsApplied{
		ok:       isOK,
		errors:   errors,
		failures: failures,
	}
}

var extractPattern, _ = regexp.Compile(`{{[a-zA-Z0-9]+=.*?}}`)

// example:
// in: "result.systemout: foo with a {{myvariable=[a-z]+}} here"
// out: "result.systemout: foo with a (?P<myvariable>[a-z]+) here"
func transformPattern(pattern string) string {
	var p = pattern
	for _, v := range extractPattern.FindAllString(pattern, -1) {
		varname := v[2:strings.Index(v, "=")]             // extract "foo from '{{foo=value}}'"
		valregex := v[strings.Index(v, "=")+1 : len(v)-2] // extract "value from '{{foo=value}}'"
		p = strings.Replace(p, "{{"+varname+"="+valregex+"}}", "(?P<"+varname+">"+valregex+")", -1)
	}

	return p
}

func checkExtracts(pattern, instring string, executorResult *ExecutorResult) (*Failure, *Failure) {
	r := regexp.MustCompile(pattern)
	match := r.FindStringSubmatch(instring)
	if match == nil {
		return &Failure{Value: RemoveNotPrintableChar(fmt.Sprintf("Pattern '%s' does not match string '%s'", pattern, instring))}, nil
	}

	e := *executorResult
	found := true
	for i, name := range r.SubexpNames() {
		if i == 0 {
			continue
		}
		e[name] = match[i]
	}

	if !found {
		return nil, &Failure{Value: RemoveNotPrintableChar(fmt.Sprintf("pattern '%s' match nothing in result '%s'", pattern, instring))}
	}
	return nil, nil
}

// RemoveNotPrintableChar removes not printable chararacter from a string
func RemoveNotPrintableChar(in string) string {
	m := func(r rune) rune {
		if unicode.IsPrint(r) || unicode.IsSpace(r) || unicode.IsPunct(r) {
			return r
		}
		return ' '
	}
	return strings.Map(m, in)
}
