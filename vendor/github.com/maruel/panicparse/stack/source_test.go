// Copyright 2015 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package stack

import (
	"bytes"
	"fmt"
	"go/ast"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maruel/ut"
)

func overrideEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = value
			return env
		}
	}
	return append(env, prefix+value)
}

func getCrash(t *testing.T, content string) (string, []byte) {
	name, err := ioutil.TempDir("", "panicparse")
	ut.AssertEqual(t, nil, err)
	defer os.RemoveAll(name)
	main := filepath.Join(name, "main.go")
	ut.AssertEqual(t, nil, ioutil.WriteFile(main, []byte(content), 0500))
	cmd := exec.Command("go", "run", main)
	// Use the Go 1.4 compatible format.
	cmd.Env = overrideEnv(os.Environ(), "GOTRACEBACK", "2")
	out, _ := cmd.CombinedOutput()
	return main, out
}

func TestAugment(t *testing.T) {
	extra := &bytes.Buffer{}
	main, content := getCrash(t, mainSource)
	goroutines, err := ParseDump(bytes.NewBuffer(content), extra)
	ut.AssertEqual(t, nil, err)
	// On go1.4, there's one less space.
	actual := extra.String()
	if actual != "panic: ooh\n\nexit status 2\n" && actual != "panic: ooh\nexit status 2\n" {
		t.Fatalf("Unexpected panic output:\n%#v", actual)
	}
	// The number of goroutine alive depends on the runtime environment. It
	// doesn't matter as only the crashing thread is of importance.
	ut.AssertEqual(t, true, len(goroutines) >= 1)

	// Preload content so no disk I/O is done.
	c := &cache{files: map[string][]byte{main: []byte(mainSource)}}
	c.augmentGoroutine(&goroutines[0])
	pointer := uint64(0xfffffffff)
	pointerStr := fmt.Sprintf("0x%x", pointer)
	expected := Stack{
		Calls: []Call{
			{
				SourcePath: filepath.Join(goroot, "src", "runtime", "panic.go"),
				Func:       Function{"panic"},
				Args:       Args{Values: []Arg{{Value: pointer}, {Value: pointer}}},
			},
			{
				Func: Function{"main.S.f1"},
			},
			{
				Func: Function{"main.(*S).f2"},
				Args: Args{
					Values:    []Arg{{Value: pointer}},
					Processed: []string{"*S(" + pointerStr + ")"},
				},
			},
			{
				Func: Function{"main.f3"},
				Args: Args{
					Values:    []Arg{{Value: pointer}, {Value: 3}, {Value: 1}},
					Processed: []string{"string(" + pointerStr + ", len=3)", "1"},
				},
			},
			{
				Func: Function{"main.f4"},
				Args: Args{
					Values:    []Arg{{Value: pointer}, {Value: 3}},
					Processed: []string{"string(" + pointerStr + ", len=3)"},
				},
			},
			{
				Func: Function{"main.f5"},
				Args: Args{
					Values:    []Arg{{}, {}, {}, {}, {}, {}, {}, {}, {}, {}},
					Processed: []string{"0", "0", "0", "0", "0", "0", "0", "0", "0", "interface{}(0x0)"},
					Elided:    true,
				},
			},
			{
				Func: Function{"main.f6"},
				Args: Args{
					Values:    []Arg{{Value: pointer}, {Value: pointer}},
					Processed: []string{"error(" + pointerStr + ")"},
				},
			},
			{
				Func: Function{"main.f7"},
				Args: Args{
					Values:    []Arg{{}, {}},
					Processed: []string{"error(0x0)"},
				},
			},
			{
				Func: Function{"main.f8"},
				Args: Args{
					Values:    []Arg{{Value: 0x3fe0000000000000}, {Value: 0xc440066666}},
					Processed: []string{"0.5", "2.1"},
				},
			},
			{
				Func: Function{"main.f9"},
				Args: Args{
					Values:    []Arg{{Value: pointer}, {Value: 5}, {Value: 7}},
					Processed: []string{"[]int(" + pointerStr + " len=5 cap=7)"},
				},
			},
			{
				Func: Function{"main.f10"},
				Args: Args{
					Values:    []Arg{{Value: pointer}, {Value: 5}, {Value: 7}},
					Processed: []string{"[]interface{}(" + pointerStr + " len=5 cap=7)"},
				},
			},
			{
				Func: Function{"main.f11"},
				Args: Args{
					Values:    []Arg{{}},
					Processed: []string{"func(0x0)"},
				},
			},
			{
				Func: Function{"main.f12"},
				Args: Args{
					Values:    []Arg{{Value: pointer}, {Value: 2}, {Value: 2}},
					Processed: []string{"func(" + pointerStr + ")", "func(0x2)"},
				},
			},
			{
				Func: Function{"main.f13"},
				Args: Args{
					Values:    []Arg{{Value: pointer}, {Value: 2}},
					Processed: []string{"string(" + pointerStr + ", len=2)"},
				},
			},
			{
				Func: Function{"main.main"},
			},
		},
	}
	s := goroutines[0].Signature.Stack
	// On Travis, runtime.GOROOT() != what is dumped when running a command via
	// "go run". E.g. GOROOT() were "/usr/local/go" yet the path output via a
	// subcommand is "/home/travis/.gimme/versions/go1.4.linux.amd64". Kidding
	// me, right?
	ut.AssertEqual(t, true, strings.HasSuffix(s.Calls[0].SourcePath, "panic.go"))
	s.Calls[0].SourcePath = expected.Calls[0].SourcePath
	// Zap out the panic() call, since its signature changed between go1.4 and
	// go1.5, it used to be runtime.gopanic().
	ut.AssertEqual(t, true, strings.HasSuffix(s.Calls[0].Func.Raw, "panic"))
	s.Calls[0].Func = expected.Calls[0].Func

	// Zap out pointers.
	for i := range s.Calls {
		if i >= len(expected.Calls) {
			// When using GOTRACEBACK=2, it'll include runtime.main() and
			// runtime.goexit(). Ignore these since they could be changed in a future
			// version.
			s.Calls = s.Calls[:len(expected.Calls)]
			break
		}
		if i > 0 {
			ut.AssertEqual(t, true, s.Calls[i].Line > s.Calls[i-1].Line)
		}
		s.Calls[i].Line = 0
		for j := range s.Calls[i].Args.Values {
			if j >= len(expected.Calls[i].Args.Values) {
				break
			}
			if expected.Calls[i].Args.Values[j].Value == pointer {
				// Replace the pointer value.
				ut.AssertEqual(t, false, s.Calls[i].Args.Values[j].Value == 0)
				old := fmt.Sprintf("0x%x", s.Calls[i].Args.Values[j].Value)
				s.Calls[i].Args.Values[j].Value = pointer
				for k := range s.Calls[i].Args.Processed {
					s.Calls[i].Args.Processed[k] = strings.Replace(s.Calls[i].Args.Processed[k], old, pointerStr, -1)
				}
			}
		}
		if expected.Calls[i].SourcePath == "" {
			expected.Calls[i].SourcePath = main
		}
	}
	// Zap out panic() exact line number.
	s.Calls[0].Line = 0

	// Zap out floating point, this is not deterministic. Verify the line # is
	// actually the right one.
	line := 8 // main.f8
	ut.AssertEqual(t, uint64(0xc440066666), expected.Calls[line].Args.Values[1].Value)
	if s.Calls[line].Args.Values[1].Value != expected.Calls[line].Args.Values[1].Value {
		// Try an alternate encoding of "2.1".
		expected.Calls[line].Args.Values[1].Value = 0x40066666
	}
	ut.AssertEqual(t, expected, s)
}

func TestAugmentDummy(t *testing.T) {
	goroutines := []Goroutine{
		{
			Signature: Signature{
				Stack: Stack{
					Calls: []Call{{SourcePath: "missing.go"}},
				},
			},
		},
	}
	Augment(goroutines)
}

func TestLoad(t *testing.T) {
	c := &cache{
		files:  map[string][]byte{"bad.go": []byte("bad content")},
		parsed: map[string]*parsedFile{},
	}
	c.load("foo.asm")
	c.load("bad.go")
	c.load("doesnt_exist.go")
	ut.AssertEqual(t, 3, len(c.parsed))
	ut.AssertEqual(t, (*parsedFile)(nil), c.parsed["foo.asm"])
	ut.AssertEqual(t, (*parsedFile)(nil), c.parsed["bad.go"])
	ut.AssertEqual(t, (*parsedFile)(nil), c.parsed["doesnt_exist.go"])
	ut.AssertEqual(t, (*ast.FuncDecl)(nil), c.getFuncAST(&Call{SourcePath: "other"}))
}

const mainSource = `// Exercises most code paths in processCall().

package main

import "errors"

type S struct {
}

func (s S) f1() {
	panic("ooh")
}

func (s *S) f2() {
	s.f1()
}

func f3(s string, i int) {
	(&S{}).f2()
}

func f4(s string) {
	f3(s, 1)
}

func f5(s1, s2, s3, s4, s5, s6, s7, s8, s9 int, s10 interface{}) {
	f4("ooh")
}

func f6(err error) {
	f5(0, 0, 0, 0, 0, 0, 0, 0, 0, nil)
}

func f7(error) {
	f6(errors.New("Ooh"))
}

func f8(a float64, b float32) {
	f7(nil)
}

func f9(a []int) {
	f8(0.5, 2.1)
}

func f10(a []interface{}) {
	f9(make([]int, 5, 7))
}

func f11(a func()) {
	f10(make([]interface{}, 5, 7))
}

func f12(a ...func()) {
	f11(nil)
}

func f13(s string) {
	// This asserts that a local function definition is not picked up by accident.
	a := func(i int) int {
		return 1 + i
	}
	_ = a(3)
	f12(nil, nil)
}

func main() {
	f13("yo")
}
`
