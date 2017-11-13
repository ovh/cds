package venom

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/mitchellh/mapstructure"
)

// applyExtracts try to run extract on step, return true if all extracts are OK, false otherwise
func applyExtracts(executorResult *ExecutorResult, step TestStep, l Logger) (bool, []Failure, []Failure) {
	var se StepExtracts
	var errors []Failure
	var failures []Failure

	if err := mapstructure.Decode(step, &se); err != nil {
		return false, []Failure{{Value: RemoveNotPrintableChar(fmt.Sprintf("error decoding extracts: %s", err))}}, failures
	}

	isOK := true
	for key, pattern := range se.Extracts {
		e := *executorResult
		if _, ok := e[key]; !ok {
			return false, []Failure{{Value: RemoveNotPrintableChar(fmt.Sprintf("key %s in result is not found", key))}}, failures
		}
		errs, fails := checkExtracts(transformPattern(pattern), fmt.Sprintf("%v", e[key]), executorResult, l)
		if errs != nil {
			errors = append(errors, *errs)
			isOK = false
		}
		if fails != nil {
			failures = append(failures, *fails)
			isOK = false
		}
	}

	return isOK, errors, failures
}

// example:
// in: "result.systemout: foo with a {{myvariable=[a-z]+}} here"
// out: "result.systemout: foo with a (?P<myvariable>[a-z]+) here"
func transformPattern(pattern string) string {
	var p = pattern

	r, _ := regexp.Compile(`{{[a-zA-Z0-9]+=.*?}}`)

	for _, v := range r.FindAllString(pattern, -1) {
		varname := v[2:strings.Index(v, "=")]             // extract "foo from '{{foo=value}}'"
		valregex := v[strings.Index(v, "=")+1 : len(v)-2] // extract "value from '{{foo=value}}'"
		p = strings.Replace(p, "{{"+varname+"="+valregex+"}}", "(?P<"+varname+">"+valregex+")", -1)
	}

	return p
}

func checkExtracts(pattern, instring string, executorResult *ExecutorResult, l Logger) (*Failure, *Failure) {
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

func RemoveNotPrintableChar(in string) string {
	m := func(r rune) rune {
		if unicode.IsPrint(r) || unicode.IsSpace(r) || unicode.IsPunct(r) {
			return r
		}
		return ' '
	}
	return strings.Map(m, in)
}
