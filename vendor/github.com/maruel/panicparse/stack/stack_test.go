// Copyright 2015 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package stack

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maruel/ut"
)

var goroot = goroots[0]

const crash = `panic: oh no!

goroutine 1 [running]:
panic(0x0, 0x0)
	/home/user/src/golang/src/runtime/panic.go:464 +0x3e6
main.crash2(0x7fe50b49d028, 0xc82000a1e0)
	/home/user/src/foo.go:45 +0x23
main.main()
	/home/user/src/foo.go:50 +0xa6
`

func Example() {
	in := bytes.NewBufferString(crash)
	goroutines, err := ParseDump(in, os.Stdout)
	if err != nil {
		return
	}

	// Optional: Check for GOTRACEBACK being set, in particular if there is only
	// one goroutine returned.

	// Use a color palette based on ANSI code.
	p := &Palette{}
	buckets := SortBuckets(Bucketize(goroutines, AnyValue))
	srcLen, pkgLen := CalcLengths(buckets, false)
	for _, bucket := range buckets {
		io.WriteString(os.Stdout, p.BucketHeader(&bucket, false, len(buckets) > 1))
		io.WriteString(os.Stdout, p.StackLines(&bucket.Signature, srcLen, pkgLen, false))
	}
	// Output:
	// panic: oh no!
	//
	// 1: running
	//          panic.go:464 panic(0, 0)
	//     main foo.go:45    crash2(0x7fe50b49d028, 0xc82000a1e0)
	//     main foo.go:50    main()
}

func TestParseDump1(t *testing.T) {
	// One call from main, one from stdlib, one from third party.
	// Create a long first line that will be ignored. It is to guard against
	// https://github.com/maruel/panicparse/issues/17.
	long := strings.Repeat("a", bufio.MaxScanTokenSize+1)
	data := []string{
		long,
		"panic: reflect.Set: value of type",
		"",
		"goroutine 1 [running]:",
		"github.com/cockroachdb/cockroach/storage/engine._Cfunc_DBIterSeek()",
		" ??:0 +0x6d",
		"gopkg.in/yaml%2ev2.handleErr(0xc208033b20)",
		"	/gopath/src/gopkg.in/yaml.v2/yaml.go:153 +0xc6",
		"reflect.Value.assignTo(0x570860, 0xc20803f3e0, 0x15)",
		"	" + goroot + "/src/reflect/value.go:2125 +0x368",
		"main.main()",
		"	/gopath/src/github.com/foo/bar/baz.go:428 +0x27",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, nil, err)
	ut.AssertEqual(t, long+"\npanic: reflect.Set: value of type\n\n", extra.String())
	expected := []Goroutine{
		{
			Signature: Signature{
				State: "running",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "??",
							Func:       Function{"github.com/cockroachdb/cockroach/storage/engine._Cfunc_DBIterSeek"},
						},
						{
							SourcePath: "/gopath/src/gopkg.in/yaml.v2/yaml.go",
							Line:       153,
							Func:       Function{"gopkg.in/yaml%2ev2.handleErr"},
							Args:       Args{Values: []Arg{{Value: 0xc208033b20}}},
						},
						{
							SourcePath: goroot + "/src/reflect/value.go",
							Line:       2125,
							Func:       Function{"reflect.Value.assignTo"},
							Args:       Args{Values: []Arg{{Value: 0x570860}, {Value: 0xc20803f3e0}, {Value: 0x15}}},
						},
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       428,
							Func:       Function{"main.main"},
						},
					},
				},
			},
			ID:    1,
			First: true,
		},
	}
	ut.AssertEqual(t, expected, goroutines)
}

func TestParseDumpLongWait(t *testing.T) {
	// One call from main, one from stdlib, one from third party.
	data := []string{
		"panic: bleh",
		"",
		"goroutine 1 [chan send, 100 minutes]:",
		"gopkg.in/yaml%2ev2.handleErr(0xc208033b20)",
		"	/gopath/src/gopkg.in/yaml.v2/yaml.go:153 +0xc6",
		"",
		"goroutine 2 [chan send, locked to thread]:",
		"gopkg.in/yaml%2ev2.handleErr(0xc208033b21)",
		"	/gopath/src/gopkg.in/yaml.v2/yaml.go:153 +0xc6",
		"",
		"goroutine 3 [chan send, 101 minutes, locked to thread]:",
		"gopkg.in/yaml%2ev2.handleErr(0xc208033b22)",
		"	/gopath/src/gopkg.in/yaml.v2/yaml.go:153 +0xc6",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, nil, err)
	ut.AssertEqual(t, "panic: bleh\n\n", extra.String())
	expected := []Goroutine{
		{
			Signature: Signature{
				State:    "chan send",
				SleepMin: 100,
				SleepMax: 100,
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/gopkg.in/yaml.v2/yaml.go",
							Line:       153,
							Func:       Function{"gopkg.in/yaml%2ev2.handleErr"},
							Args:       Args{Values: []Arg{{Value: 0xc208033b20}}},
						},
					},
				},
			},
			ID:    1,
			First: true,
		},
		{
			Signature: Signature{
				State:  "chan send",
				Locked: true,
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/gopkg.in/yaml.v2/yaml.go",
							Line:       153,
							Func:       Function{"gopkg.in/yaml%2ev2.handleErr"},
							Args:       Args{Values: []Arg{{Value: 0xc208033b21, Name: "#1"}}},
						},
					},
				},
			},
			ID: 2,
		},
		{
			Signature: Signature{
				State:    "chan send",
				SleepMin: 101,
				SleepMax: 101,
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/gopkg.in/yaml.v2/yaml.go",
							Line:       153,
							Func:       Function{"gopkg.in/yaml%2ev2.handleErr"},
							Args:       Args{Values: []Arg{{Value: 0xc208033b22, Name: "#2"}}},
						},
					},
				},
				Locked: true,
			},
			ID: 3,
		},
	}
	ut.AssertEqual(t, expected, goroutines)
}

func TestParseDumpAsm(t *testing.T) {
	data := []string{
		"panic: reflect.Set: value of type",
		"",
		"goroutine 16 [garbage collection]:",
		"runtime.switchtoM()",
		"\t" + goroot + "/src/runtime/asm_amd64.s:198 fp=0xc20cfb80d8 sp=0xc20cfb80d0",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, nil, err)
	expected := []Goroutine{
		{
			Signature: Signature{
				State: "garbage collection",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: goroot + "/src/runtime/asm_amd64.s",
							Line:       198,
							Func:       Function{Raw: "runtime.switchtoM"},
						},
					},
				},
			},
			ID:    16,
			First: true,
		},
	}
	ut.AssertEqual(t, expected, goroutines)
	ut.AssertEqual(t, "panic: reflect.Set: value of type\n\n", extra.String())
}

func TestParseDumpLineErr(t *testing.T) {
	data := []string{
		"panic: reflect.Set: value of type",
		"",
		"goroutine 1 [running]:",
		"github.com/foo/bar.recurseType()",
		"\t/gopath/src/github.com/foo/bar/baz.go:12345678901234567890",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, errors.New("failed to parse int on line: \"\t/gopath/src/github.com/foo/bar/baz.go:12345678901234567890\n\""), err)
	expected := []Goroutine{
		{
			Signature: Signature{
				State: "running",
				Stack: Stack{Calls: []Call{{Func: Function{Raw: "github.com/foo/bar.recurseType"}}}},
			},
			ID:    1,
			First: true,
		},
	}

	ut.AssertEqual(t, expected, goroutines)
}

func TestParseDumpValueErr(t *testing.T) {
	data := []string{
		"panic: reflect.Set: value of type",
		"",
		"goroutine 1 [running]:",
		"github.com/foo/bar.recurseType(123456789012345678901)",
		"\t/gopath/src/github.com/foo/bar/baz.go:9",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, errors.New("failed to parse int on line: \"github.com/foo/bar.recurseType(123456789012345678901)\n\""), err)
	expected := []Goroutine{
		{
			Signature: Signature{State: "running"},
			ID:        1,
			First:     true,
		},
	}

	ut.AssertEqual(t, expected, goroutines)
}

func TestParseDumpOrderErr(t *testing.T) {
	data := []string{
		"panic: reflect.Set: value of type",
		"",
		"goroutine 16 [garbage collection]:",
		"	/gopath/src/gopkg.in/yaml.v2/yaml.go:153 +0xc6",
		"runtime.switchtoM()",
		"\t" + goroot + "/src/runtime/asm_amd64.s:198 fp=0xc20cfb80d8 sp=0xc20cfb80d0",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, errors.New("unexpected order"), err)
	expected := []Goroutine{
		{
			Signature: Signature{State: "garbage collection"},
			ID:        16,
			First:     true,
		},
	}
	ut.AssertEqual(t, expected, goroutines)
	ut.AssertEqual(t, "panic: reflect.Set: value of type\n\n", extra.String())
}

func TestParseDumpElided(t *testing.T) {
	data := []string{
		"panic: reflect.Set: value of type",
		"",
		"goroutine 16 [garbage collection]:",
		"github.com/foo/bar.recurseType(0x7f4fa9a3ec70, 0xc208062580, 0x7f4fa9a3e818, 0x50a820, 0xc20803a8a0)",
		"\t/gopath/src/github.com/foo/bar/baz.go:53 +0x845 fp=0xc20cfc66d8 sp=0xc20cfc6470",
		"...additional frames elided...",
		"created by testing.RunTests",
		"\t" + goroot + "/src/testing/testing.go:555 +0xa8b",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, nil, err)
	expected := []Goroutine{
		{
			Signature: Signature{
				State: "garbage collection",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       53,
							Func:       Function{Raw: "github.com/foo/bar.recurseType"},
							Args: Args{
								Values: []Arg{
									{Value: 0x7f4fa9a3ec70},
									{Value: 0xc208062580},
									{Value: 0x7f4fa9a3e818},
									{Value: 0x50a820},
									{Value: 0xc20803a8a0},
								},
							},
						},
					},
					Elided: true,
				},
				CreatedBy: Call{
					SourcePath: goroot + "/src/testing/testing.go",
					Line:       555,
					Func:       Function{Raw: "testing.RunTests"},
				},
			},
			ID:    16,
			First: true,
		},
	}
	ut.AssertEqual(t, expected, goroutines)
	ut.AssertEqual(t, "panic: reflect.Set: value of type\n\n", extra.String())
}

func TestParseDumpSysCall(t *testing.T) {
	data := []string{
		"panic: reflect.Set: value of type",
		"",
		"goroutine 5 [syscall]:",
		"runtime.notetsleepg(0x918100, 0xffffffffffffffff, 0x1)",
		"\t" + goroot + "/src/runtime/lock_futex.go:201 +0x52 fp=0xc208018f68 sp=0xc208018f40",
		"runtime.signal_recv(0x0)",
		"\t" + goroot + "/src/runtime/sigqueue.go:109 +0x135 fp=0xc208018fa0 sp=0xc208018f68",
		"os/signal.loop()",
		"\t" + goroot + "/src/os/signal/signal_unix.go:21 +0x1f fp=0xc208018fe0 sp=0xc208018fa0",
		"runtime.goexit()",
		"\t" + goroot + "/src/runtime/asm_amd64.s:2232 +0x1 fp=0xc208018fe8 sp=0xc208018fe0",
		"created by os/signal.init·1",
		"\t" + goroot + "/src/os/signal/signal_unix.go:27 +0x35",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, nil, err)
	expected := []Goroutine{
		{
			Signature: Signature{
				State: "syscall",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: goroot + "/src/runtime/lock_futex.go",
							Line:       201,
							Func:       Function{Raw: "runtime.notetsleepg"},
							Args: Args{
								Values: []Arg{
									{Value: 0x918100},
									{Value: 0xffffffffffffffff},
									{Value: 0x1},
								},
							},
						},
						{
							SourcePath: goroot + "/src/runtime/sigqueue.go",
							Line:       109,
							Func:       Function{Raw: "runtime.signal_recv"},
							Args: Args{
								Values: []Arg{{}},
							},
						},
						{
							SourcePath: goroot + "/src/os/signal/signal_unix.go",
							Line:       21,
							Func:       Function{Raw: "os/signal.loop"},
						},
						{
							SourcePath: goroot + "/src/runtime/asm_amd64.s",
							Line:       2232,
							Func:       Function{Raw: "runtime.goexit"},
						},
					},
				},
				CreatedBy: Call{
					SourcePath: goroot + "/src/os/signal/signal_unix.go",
					Line:       27,
					Func:       Function{Raw: "os/signal.init·1"},
				},
			},
			ID:    5,
			First: true,
		},
	}
	ut.AssertEqual(t, expected, goroutines)
	ut.AssertEqual(t, "panic: reflect.Set: value of type\n\n", extra.String())
}

func TestParseDumpUnavail(t *testing.T) {
	data := []string{
		"panic: reflect.Set: value of type",
		"",
		"goroutine 24 [running]:",
		"\tgoroutine running on other thread; stack unavailable",
		"created by github.com/foo.New",
		"\t/gopath/src/github.com/foo/bar.go:131 +0x381",
		"",
	}
	extra := &bytes.Buffer{}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), extra)
	ut.AssertEqual(t, nil, err)
	expected := []Goroutine{
		{
			Signature: Signature{
				State: "running",
				Stack: Stack{
					Calls: []Call{{SourcePath: "<unavailable>"}},
				},
				CreatedBy: Call{
					SourcePath: "/gopath/src/github.com/foo/bar.go",
					Line:       131,
					Func:       Function{Raw: "github.com/foo.New"},
				},
			},
			ID:    24,
			First: true,
		},
	}
	ut.AssertEqual(t, expected, goroutines)
	ut.AssertEqual(t, "panic: reflect.Set: value of type\n\n", extra.String())
}

func TestParseDumpSameBucket(t *testing.T) {
	// 2 goroutines with the same signature
	data := []string{
		"panic: runtime error: index out of range",
		"",
		"goroutine 6 [chan receive]:",
		"main.func·001()",
		"	/gopath/src/github.com/foo/bar/baz.go:72 +0x49",
		"created by main.mainImpl",
		"	/gopath/src/github.com/foo/bar/baz.go:74 +0xeb",
		"",
		"goroutine 7 [chan receive]:",
		"main.func·001()",
		"	/gopath/src/github.com/foo/bar/baz.go:72 +0x49",
		"created by main.mainImpl",
		"	/gopath/src/github.com/foo/bar/baz.go:74 +0xeb",
		"",
	}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), &bytes.Buffer{})
	ut.AssertEqual(t, nil, err)
	expectedGR := []Goroutine{
		{
			Signature: Signature{
				State: "chan receive",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       72,
							Func:       Function{"main.func·001"},
						},
					},
				},
				CreatedBy: Call{
					SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
					Line:       74,
					Func:       Function{"main.mainImpl"},
				},
			},
			ID:    6,
			First: true,
		},
		{
			Signature: Signature{
				State: "chan receive",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       72,
							Func:       Function{"main.func·001"},
						},
					},
				},
				CreatedBy: Call{
					SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
					Line:       74,
					Func:       Function{"main.mainImpl"},
				},
			},
			ID: 7,
		},
	}
	ut.AssertEqual(t, expectedGR, goroutines)
	expectedBuckets := Buckets{{expectedGR[0].Signature, []Goroutine{expectedGR[0], expectedGR[1]}}}
	ut.AssertEqual(t, expectedBuckets, SortBuckets(Bucketize(goroutines, ExactLines)))
}

func TestBucketizeNotAggressive(t *testing.T) {
	// 2 goroutines with the same signature
	data := []string{
		"panic: runtime error: index out of range",
		"",
		"goroutine 6 [chan receive]:",
		"main.func·001(0x11000000, 2)",
		"	/gopath/src/github.com/foo/bar/baz.go:72 +0x49",
		"",
		"goroutine 7 [chan receive]:",
		"main.func·001(0x21000000, 2)",
		"	/gopath/src/github.com/foo/bar/baz.go:72 +0x49",
		"",
	}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), &bytes.Buffer{})
	ut.AssertEqual(t, nil, err)
	expectedGR := []Goroutine{
		{
			Signature: Signature{
				State: "chan receive",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       72,
							Func:       Function{"main.func·001"},
							Args:       Args{Values: []Arg{{0x11000000, ""}, {Value: 2}}},
						},
					},
				},
			},
			ID:    6,
			First: true,
		},
		{
			Signature: Signature{
				State: "chan receive",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       72,
							Func:       Function{"main.func·001"},
							Args:       Args{Values: []Arg{{0x21000000, "#1"}, {Value: 2}}},
						},
					},
				},
			},
			ID: 7,
		},
	}
	ut.AssertEqual(t, expectedGR, goroutines)
	expectedBuckets := Buckets{
		{expectedGR[0].Signature, []Goroutine{expectedGR[0]}},
		{expectedGR[1].Signature, []Goroutine{expectedGR[1]}},
	}
	ut.AssertEqual(t, expectedBuckets, SortBuckets(Bucketize(goroutines, ExactLines)))
}

func TestBucketizeAggressive(t *testing.T) {
	// 2 goroutines with the same signature
	data := []string{
		"panic: runtime error: index out of range",
		"",
		"goroutine 6 [chan receive, 10 minutes]:",
		"main.func·001(0x11000000, 2)",
		"	/gopath/src/github.com/foo/bar/baz.go:72 +0x49",
		"",
		"goroutine 7 [chan receive, 50 minutes]:",
		"main.func·001(0x21000000, 2)",
		"	/gopath/src/github.com/foo/bar/baz.go:72 +0x49",
		"",
		"goroutine 8 [chan receive, 100 minutes]:",
		"main.func·001(0x21000000, 2)",
		"	/gopath/src/github.com/foo/bar/baz.go:72 +0x49",
		"",
	}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), &bytes.Buffer{})
	ut.AssertEqual(t, nil, err)
	expectedGR := []Goroutine{
		{
			Signature: Signature{
				State:    "chan receive",
				SleepMin: 10,
				SleepMax: 10,
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       72,
							Func:       Function{"main.func·001"},
							Args:       Args{Values: []Arg{{0x11000000, ""}, {Value: 2}}},
						},
					},
				},
			},
			ID:    6,
			First: true,
		},
		{
			Signature: Signature{
				State:    "chan receive",
				SleepMin: 50,
				SleepMax: 50,
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       72,
							Func:       Function{"main.func·001"},
							Args:       Args{Values: []Arg{{0x21000000, "#1"}, {Value: 2}}},
						},
					},
				},
			},
			ID: 7,
		},
		{
			Signature: Signature{
				State:    "chan receive",
				SleepMin: 100,
				SleepMax: 100,
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
							Line:       72,
							Func:       Function{"main.func·001"},
							Args:       Args{Values: []Arg{{0x21000000, "#1"}, {Value: 2}}},
						},
					},
				},
			},
			ID: 8,
		},
	}
	ut.AssertEqual(t, expectedGR, goroutines)
	signature := Signature{
		State:    "chan receive",
		SleepMin: 10,
		SleepMax: 100,
		Stack: Stack{
			Calls: []Call{
				{
					SourcePath: "/gopath/src/github.com/foo/bar/baz.go",
					Line:       72,
					Func:       Function{"main.func·001"},
					Args:       Args{Values: []Arg{{0x11000000, "*"}, {Value: 2}}},
				},
			},
		},
	}
	expectedBuckets := Buckets{{signature, []Goroutine{expectedGR[0], expectedGR[1], expectedGR[2]}}}
	ut.AssertEqual(t, expectedBuckets, SortBuckets(Bucketize(goroutines, AnyPointer)))
}

func TestParseDumpNoOffset(t *testing.T) {
	data := []string{
		"panic: runtime error: index out of range",
		"",
		"goroutine 37 [runnable]:",
		"github.com/foo.func·002()",
		"	/gopath/src/github.com/foo/bar.go:110",
		"created by github.com/foo.New",
		"	/gopath/src/github.com/foo/bar.go:113 +0x43b",
		"",
	}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), &bytes.Buffer{})
	ut.AssertEqual(t, nil, err)
	expectedGR := []Goroutine{
		{
			Signature: Signature{
				State: "runnable",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: "/gopath/src/github.com/foo/bar.go",
							Line:       110,
							Func:       Function{"github.com/foo.func·002"},
						},
					},
				},
				CreatedBy: Call{
					SourcePath: "/gopath/src/github.com/foo/bar.go",
					Line:       113,
					Func:       Function{"github.com/foo.New"},
				},
			},
			ID:    37,
			First: true,
		},
	}
	ut.AssertEqual(t, expectedGR, goroutines)
}

func TestParseDumpJunk(t *testing.T) {
	// For coverage of scanLines.
	data := []string{
		"panic: reflect.Set: value of type",
		"",
		"goroutine 1 [running]:",
		"junk",
	}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), &bytes.Buffer{})
	ut.AssertEqual(t, nil, err)
	expectedGR := []Goroutine{
		{
			Signature: Signature{State: "running"},
			ID:        1,
			First:     true,
		},
	}
	ut.AssertEqual(t, expectedGR, goroutines)
}

func TestParseCCode(t *testing.T) {
	data := []string{
		"SIGQUIT: quit",
		"PC=0x43f349",
		"",
		"goroutine 0 [idle]:",
		"runtime.epollwait(0x4, 0x7fff671c7118, 0xffffffff00000080, 0x0, 0xffffffff0028c1be, 0x0, 0x0, 0x0, 0x0, 0x0, ...)",
		"        " + goroot + "/src/runtime/sys_linux_amd64.s:400 +0x19",
		"runtime.netpoll(0x901b01, 0x0)",
		"        " + goroot + "/src/runtime/netpoll_epoll.go:68 +0xa3",
		"findrunnable(0xc208012000)",
		"        " + goroot + "/src/runtime/proc.c:1472 +0x485",
		"schedule()",
		"        " + goroot + "/src/runtime/proc.c:1575 +0x151",
		"runtime.park_m(0xc2080017a0)",
		"        " + goroot + "/src/runtime/proc.c:1654 +0x113",
		"runtime.mcall(0x432684)",
		"        " + goroot + "/src/runtime/asm_amd64.s:186 +0x5a",
		"",
	}
	goroutines, err := ParseDump(bytes.NewBufferString(strings.Join(data, "\n")), &bytes.Buffer{})
	ut.AssertEqual(t, nil, err)
	expectedGR := []Goroutine{
		{
			Signature: Signature{
				State: "idle",
				Stack: Stack{
					Calls: []Call{
						{
							SourcePath: goroot + "/src/runtime/sys_linux_amd64.s",
							Line:       400,
							Func:       Function{"runtime.epollwait"},
							Args: Args{
								Values: []Arg{
									{Value: 0x4},
									{Value: 0x7fff671c7118},
									{Value: 0xffffffff00000080},
									{},
									{Value: 0xffffffff0028c1be},
									{},
									{},
									{},
									{},
									{},
								},
								Elided: true,
							},
						},
						{
							SourcePath: goroot + "/src/runtime/netpoll_epoll.go",
							Line:       68,
							Func:       Function{"runtime.netpoll"},
							Args:       Args{Values: []Arg{{Value: 0x901b01}, {}}},
						},
						{
							SourcePath: goroot + "/src/runtime/proc.c",
							Line:       1472,
							Func:       Function{"findrunnable"},
							Args:       Args{Values: []Arg{{Value: 0xc208012000}}},
						},
						{
							SourcePath: goroot + "/src/runtime/proc.c",
							Line:       1575,
							Func:       Function{"schedule"},
						},
						{
							SourcePath: goroot + "/src/runtime/proc.c",
							Line:       1654,
							Func:       Function{"runtime.park_m"},
							Args:       Args{Values: []Arg{{Value: 0xc2080017a0}}},
						},
						{
							SourcePath: goroot + "/src/runtime/asm_amd64.s",
							Line:       186,
							Func:       Function{"runtime.mcall"},
							Args:       Args{Values: []Arg{{Value: 0x432684}}},
						},
					},
				},
			},
			ID:    0,
			First: true,
		},
	}
	ut.AssertEqual(t, expectedGR, goroutines)
}

func TestCallPkg1(t *testing.T) {
	c := Call{
		SourcePath: "/gopath/src/gopkg.in/yaml.v2/yaml.go",
		Line:       153,
		Func:       Function{"gopkg.in/yaml%2ev2.handleErr"},
		Args:       Args{Values: []Arg{{Value: 0xc208033b20}}},
	}
	ut.AssertEqual(t, "yaml.go", c.SourceName())
	ut.AssertEqual(t, filepath.Join("yaml.v2", "yaml.go"), c.PkgSource())
	ut.AssertEqual(t, "gopkg.in/yaml.v2.handleErr", c.Func.String())
	ut.AssertEqual(t, "handleErr", c.Func.Name())
	// This is due to directory name not matching the package name.
	ut.AssertEqual(t, "yaml.v2", c.Func.PkgName())
	ut.AssertEqual(t, false, c.Func.IsExported())
	ut.AssertEqual(t, false, c.IsStdlib())
	ut.AssertEqual(t, false, c.IsPkgMain())
}

func TestCallPkg2(t *testing.T) {
	c := Call{
		SourcePath: "/gopath/src/gopkg.in/yaml.v2/yaml.go",
		Line:       153,
		Func:       Function{"gopkg.in/yaml%2ev2.(*decoder).unmarshal"},
		Args:       Args{Values: []Arg{{Value: 0xc208033b20}}},
	}
	ut.AssertEqual(t, "yaml.go", c.SourceName())
	ut.AssertEqual(t, filepath.Join("yaml.v2", "yaml.go"), c.PkgSource())
	// TODO(maruel): Using '/' for this function is inconsistent on Windows
	// w.r.t. other functions.
	ut.AssertEqual(t, "gopkg.in/yaml.v2.(*decoder).unmarshal", c.Func.String())
	ut.AssertEqual(t, "(*decoder).unmarshal", c.Func.Name())
	// This is due to directory name not matching the package name.
	ut.AssertEqual(t, "yaml.v2", c.Func.PkgName())
	ut.AssertEqual(t, false, c.Func.IsExported())
	ut.AssertEqual(t, false, c.IsStdlib())
	ut.AssertEqual(t, false, c.IsPkgMain())
}

func TestCallStdlib(t *testing.T) {
	c := Call{
		SourcePath: goroot + "/src/reflect/value.go",
		Line:       2125,
		Func:       Function{"reflect.Value.assignTo"},
		Args:       Args{Values: []Arg{{Value: 0x570860}, {Value: 0xc20803f3e0}, {Value: 0x15}}},
	}
	ut.AssertEqual(t, "value.go", c.SourceName())
	ut.AssertEqual(t, "value.go:2125", c.SourceLine())
	ut.AssertEqual(t, filepath.Join("reflect", "value.go"), c.PkgSource())
	ut.AssertEqual(t, "reflect.Value.assignTo", c.Func.String())
	ut.AssertEqual(t, "Value.assignTo", c.Func.Name())
	ut.AssertEqual(t, "reflect", c.Func.PkgName())
	ut.AssertEqual(t, false, c.Func.IsExported())
	ut.AssertEqual(t, true, c.IsStdlib())
	ut.AssertEqual(t, false, c.IsPkgMain())
}

func TestCallMain(t *testing.T) {
	c := Call{
		SourcePath: "/gopath/src/github.com/foo/bar/main.go",
		Line:       428,
		Func:       Function{"main.main"},
	}
	ut.AssertEqual(t, "main.go", c.SourceName())
	ut.AssertEqual(t, "main.go:428", c.SourceLine())
	ut.AssertEqual(t, filepath.Join("bar", "main.go"), c.PkgSource())
	ut.AssertEqual(t, "main.main", c.Func.String())
	ut.AssertEqual(t, "main", c.Func.Name())
	ut.AssertEqual(t, "main", c.Func.PkgName())
	ut.AssertEqual(t, true, c.Func.IsExported())
	ut.AssertEqual(t, false, c.IsStdlib())
	ut.AssertEqual(t, true, c.IsPkgMain())
}

func TestCallC(t *testing.T) {
	c := Call{
		SourcePath: goroot + "/src/runtime/proc.c",
		Line:       1472,
		Func:       Function{"findrunnable"},
		Args:       Args{Values: []Arg{{Value: 0xc208012000}}},
	}
	ut.AssertEqual(t, "proc.c", c.SourceName())
	ut.AssertEqual(t, "proc.c:1472", c.SourceLine())
	ut.AssertEqual(t, filepath.Join("runtime", "proc.c"), c.PkgSource())
	ut.AssertEqual(t, "findrunnable", c.Func.String())
	ut.AssertEqual(t, "findrunnable", c.Func.Name())
	ut.AssertEqual(t, "", c.Func.PkgName())
	ut.AssertEqual(t, false, c.Func.IsExported())
	ut.AssertEqual(t, true, c.IsStdlib())
	ut.AssertEqual(t, false, c.IsPkgMain())
}

func TestArgs(t *testing.T) {
	a := Args{
		Values: []Arg{
			{Value: 0x4},
			{Value: 0x7fff671c7118},
			{Value: 0xffffffff00000080},
			{},
			{Value: 0xffffffff0028c1be},
			{},
			{},
			{},
			{},
			{},
		},
		Elided: true,
	}
	ut.AssertEqual(t, "0x4, 0x7fff671c7118, 0xffffffff00000080, 0, 0xffffffff0028c1be, 0, 0, 0, 0, 0, ...", a.String())
}

func TestFunctionAnonymous(t *testing.T) {
	f := Function{"main.func·001"}
	ut.AssertEqual(t, "main.func·001", f.String())
	ut.AssertEqual(t, "main.func·001", f.PkgDotName())
	ut.AssertEqual(t, "func·001", f.Name())
	ut.AssertEqual(t, "main", f.PkgName())
	ut.AssertEqual(t, false, f.IsExported())
}

func TestFunctionGC(t *testing.T) {
	f := Function{"gc"}
	ut.AssertEqual(t, "gc", f.String())
	ut.AssertEqual(t, "gc", f.PkgDotName())
	ut.AssertEqual(t, "gc", f.Name())
	ut.AssertEqual(t, "", f.PkgName())
	ut.AssertEqual(t, false, f.IsExported())
}
