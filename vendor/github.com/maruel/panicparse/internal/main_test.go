// Copyright 2015 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package internal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/maruel/panicparse/stack"
	"github.com/maruel/ut"
)

var data = []string{
	"panic: runtime error: index out of range",
	"",
	"goroutine 11 [running, 5 minutes, locked to thread]:",
	"github.com/luci/luci-go/client/archiver.(*archiver).PushFile(0xc208032410, 0xc20968a3c0, 0x5b, 0xc20988c280, 0x7d, 0x0, 0x0)",
	"        /gopath/path/to/archiver.go:325 +0x2c4",
	"github.com/luci/luci-go/client/isolate.archive(0x7fbdab7a5218, 0xc208032410, 0xc20803b0b0, 0x22, 0xc208046370, 0xc20804666a, 0x17, 0x0, 0x0, 0x0, ...)",
	"        /gopath/path/to/isolate.go:148 +0x12d2",
	"github.com/luci/luci-go/client/isolate.Archive(0x7fbdab7a5218, 0xc208032410, 0xc20803b0b0, 0x22, 0xc208046370, 0x0, 0x0)",
	"        /gopath/path/to/isolate.go:102 +0xc9",
	"main.func·004(0x7fffc3b8f13a, 0x2c)",
	"        /gopath/path/to/batch_archive.go:166 +0x7cd",
	"created by main.(*batchArchiveRun).main",
	"        /gopath/path/to/batch_archive.go:167 +0x42c",
	"",
	"goroutine 1 [running]:",
	"gopkg.in/yaml%2ev2.handleErr(0xc208033b20)",
	" /gopath/src/gopkg.in/yaml.v2/yaml.go:153 +0xc6",
	"reflect.Value.assignTo(0x570860, 0xc20803f3e0, 0x15)",
	" c:/go/src/reflect/value.go:2125 +0x368",
	"main.main()",
	" /gopath/src/github.com/maruel/pre-commit-go/main.go:428 +0x27",
	"",
	"goroutine 2 [running, 1 minutes]:",
	"gopkg.in/yaml%2ev2.handleErr(0xc208033b20)",
	" /gopath/src/gopkg.in/yaml.v2/yaml.go:153 +0xc6",
	"reflect.Value.assignTo(0x570860, 0xc20803f3e0, 0x15)",
	" c:/go/src/reflect/value.go:2125 +0x368",
	"main.main()",
	" /gopath/src/github.com/maruel/pre-commit-go/main.go:428 +0x27",
	"",
}

func TestProcess(t *testing.T) {
	out := &bytes.Buffer{}
	err := process(bytes.NewBufferString(strings.Join(data, "\n")), out, &defaultPalette, stack.AnyPointer, false, false)
	ut.AssertEqual(t, nil, err)
	expected := []string{
		"panic: runtime error: index out of range",
		"",
		"\x1b[1;35m1: running [5 minutes] [locked]\x1b[90m [Created by main.(*batchArchiveRun).main @ batch_archive.go:167]\x1b[39m\x1b[m",
		"    \x1b[1;39marchiver \x1b[39m\x1b[marchiver.go:325      \x1b[1;31m(*archiver).PushFile\x1b[39m\x1b[m(#1, 0xc20968a3c0, 0x5b, 0xc20988c280, 0x7d, 0, 0)\x1b[39m\x1b[m",
		"    \x1b[1;39misolate  \x1b[39m\x1b[misolate.go:148       \x1b[31marchive\x1b[39m\x1b[m(#4, #1, #2, 0x22, #3, 0xc20804666a, 0x17, 0, 0, 0, ...)\x1b[39m\x1b[m",
		"    \x1b[1;39misolate  \x1b[39m\x1b[misolate.go:102       \x1b[1;31mArchive\x1b[39m\x1b[m(#4, #1, #2, 0x22, #3, 0, 0)\x1b[39m\x1b[m",
		"    \x1b[1;39mmain     \x1b[39m\x1b[mbatch_archive.go:166 \x1b[1;33mfunc·004\x1b[39m\x1b[m(0x7fffc3b8f13a, 0x2c)\x1b[39m\x1b[m",
		"2: running [0~1 minutes]\x1b[39m\x1b[m",
		"    \x1b[1;39myaml.v2  \x1b[39m\x1b[myaml.go:153          \x1b[31mhandleErr\x1b[39m\x1b[m(#5)\x1b[39m\x1b[m",
		"    \x1b[1;39mreflect  \x1b[39m\x1b[mvalue.go:2125        \x1b[32mValue.assignTo\x1b[39m\x1b[m(0x570860, #6, 0x15)\x1b[39m\x1b[m",
		"    \x1b[1;39mmain     \x1b[39m\x1b[mmain.go:428          \x1b[1;33mmain\x1b[39m\x1b[m()\x1b[39m\x1b[m",
		"",
	}
	actual := strings.Split(out.String(), "\n")
	for i := 0; i < len(actual) && i < len(expected); i++ {
		ut.AssertEqualIndex(t, i, expected[i], actual[i])
	}
	ut.AssertEqual(t, expected, actual)
}

func TestProcessFullPath(t *testing.T) {
	out := &bytes.Buffer{}
	err := process(bytes.NewBufferString(strings.Join(data, "\n")), out, &defaultPalette, stack.AnyValue, true, false)
	ut.AssertEqual(t, nil, err)
	expected := []string{
		"panic: runtime error: index out of range",
		"",
		"\x1b[1;35m1: running [5 minutes] [locked]\x1b[90m [Created by main.(*batchArchiveRun).main @ /gopath/path/to/batch_archive.go:167]\x1b[39m\x1b[m",
		"    \x1b[1;39marchiver \x1b[39m\x1b[m/gopath/path/to/archiver.go:325                         \x1b[1;31m(*archiver).PushFile\x1b[39m\x1b[m(#1, 0xc20968a3c0, 0x5b, 0xc20988c280, 0x7d, 0, 0)\x1b[39m\x1b[m",
		"    \x1b[1;39misolate  \x1b[39m\x1b[m/gopath/path/to/isolate.go:148                          \x1b[31marchive\x1b[39m\x1b[m(#4, #1, #2, 0x22, #3, 0xc20804666a, 0x17, 0, 0, 0, ...)\x1b[39m\x1b[m",
		"    \x1b[1;39misolate  \x1b[39m\x1b[m/gopath/path/to/isolate.go:102                          \x1b[1;31mArchive\x1b[39m\x1b[m(#4, #1, #2, 0x22, #3, 0, 0)\x1b[39m\x1b[m",
		"    \x1b[1;39mmain     \x1b[39m\x1b[m/gopath/path/to/batch_archive.go:166                    \x1b[1;33mfunc·004\x1b[39m\x1b[m(0x7fffc3b8f13a, 0x2c)\x1b[39m\x1b[m",
		"2: running [0~1 minutes]\x1b[39m\x1b[m",
		"    \x1b[1;39myaml.v2  \x1b[39m\x1b[m/gopath/src/gopkg.in/yaml.v2/yaml.go:153                \x1b[31mhandleErr\x1b[39m\x1b[m(#5)\x1b[39m\x1b[m",
		"    \x1b[1;39mreflect  \x1b[39m\x1b[mc:/go/src/reflect/value.go:2125                         \x1b[32mValue.assignTo\x1b[39m\x1b[m(0x570860, #6, 0x15)\x1b[39m\x1b[m",
		"    \x1b[1;39mmain     \x1b[39m\x1b[m/gopath/src/github.com/maruel/pre-commit-go/main.go:428 \x1b[1;33mmain\x1b[39m\x1b[m()\x1b[39m\x1b[m",
		"",
	}
	actual := strings.Split(out.String(), "\n")
	for i := 0; i < len(actual) && i < len(expected); i++ {
		ut.AssertEqualIndex(t, i, expected[i], actual[i])
	}
	ut.AssertEqual(t, expected, actual)
}

func TestProcessNoColor(t *testing.T) {
	out := &bytes.Buffer{}
	err := process(bytes.NewBufferString(strings.Join(data, "\n")), out, &stack.Palette{}, stack.AnyPointer, false, false)
	ut.AssertEqual(t, nil, err)
	expected := []string{
		"panic: runtime error: index out of range",
		"",
		"1: running [5 minutes] [locked] [Created by main.(*batchArchiveRun).main @ batch_archive.go:167]",
		"    archiver archiver.go:325      (*archiver).PushFile(#1, 0xc20968a3c0, 0x5b, 0xc20988c280, 0x7d, 0, 0)",
		"    isolate  isolate.go:148       archive(#4, #1, #2, 0x22, #3, 0xc20804666a, 0x17, 0, 0, 0, ...)",
		"    isolate  isolate.go:102       Archive(#4, #1, #2, 0x22, #3, 0, 0)",
		"    main     batch_archive.go:166 func·004(0x7fffc3b8f13a, 0x2c)",
		"2: running [0~1 minutes]",
		"    yaml.v2  yaml.go:153          handleErr(#5)",
		"    reflect  value.go:2125        Value.assignTo(0x570860, #6, 0x15)",
		"    main     main.go:428          main()",
		"",
	}
	actual := strings.Split(out.String(), "\n")
	for i := 0; i < len(actual) && i < len(expected); i++ {
		ut.AssertEqualIndex(t, i, expected[i], actual[i])
	}
	ut.AssertEqual(t, expected, actual)
}
