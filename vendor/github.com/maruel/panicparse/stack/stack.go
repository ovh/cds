// Copyright 2015 Marc-Antoine Ruel. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package stack analyzes stack dump of Go processes and simplifies it.
//
// It is mostly useful on servers will large number of identical goroutines,
// making the crash dump harder to read than strictly necessary.
package stack

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Func is a function call.
//
// Go stack traces print a mangled function call, this wrapper unmangle the
// string before printing and adds other filtering methods.
type Func struct {
	Raw string
}

// String is the fully qualified function name.
//
// Sadly Go is a bit confused when the package name doesn't match the directory
// containing the source file and will use the directory name instead of the
// real package name.
func (f *Func) String() string {
	s, _ := url.QueryUnescape(f.Raw)
	return s
}

// Name is the naked function name.
func (f *Func) Name() string {
	parts := strings.SplitN(filepath.Base(f.Raw), ".", 2)
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[1]
}

// PkgName is the package name for this function reference.
func (f *Func) PkgName() string {
	parts := strings.SplitN(filepath.Base(f.Raw), ".", 2)
	if len(parts) == 1 {
		return ""
	}
	s, _ := url.QueryUnescape(parts[0])
	return s
}

// PkgDotName returns "<package>.<func>" format.
func (f *Func) PkgDotName() string {
	parts := strings.SplitN(filepath.Base(f.Raw), ".", 2)
	s, _ := url.QueryUnescape(parts[0])
	if len(parts) == 1 {
		return parts[0]
	}
	if s != "" || parts[1] != "" {
		return s + "." + parts[1]
	}
	return ""
}

// IsExported returns true if the function is exported.
func (f *Func) IsExported() bool {
	name := f.Name()
	parts := strings.Split(name, ".")
	r, _ := utf8.DecodeRuneInString(parts[len(parts)-1])
	if unicode.ToUpper(r) == r {
		return true
	}
	return f.PkgName() == "main" && name == "main"
}

// Arg is an argument on a Call.
type Arg struct {
	Value uint64 // Value is the raw value as found in the stack trace
	Name  string // Name is a pseudo name given to the argument
}

// IsPtr returns true if we guess it's a pointer. It's only a guess, it can be
// easily be confused by a bitmask.
func (a *Arg) IsPtr() bool {
	// Assumes all pointers are above 16Mb and positive.
	return a.Value > 16*1024*1024 && a.Value < math.MaxInt64
}

func (a *Arg) String() string {
	if a.Name != "" {
		return a.Name
	}
	if a.Value == 0 {
		return "0"
	}
	return fmt.Sprintf("0x%x", a.Value)
}

// Args is a series of function call arguments.
type Args struct {
	Values    []Arg    // Values is the arguments as shown on the stack trace. They are mangled via simplification.
	Processed []string // Processed is the arguments generated from processing the source files. It can have a length lower than Values.
	Elided    bool     // If set, it means there was a trailing ", ..."
}

func (a *Args) String() string {
	var v []string
	if len(a.Processed) != 0 {
		v = make([]string, 0, len(a.Processed))
		for _, item := range a.Processed {
			v = append(v, item)
		}
	} else {
		v = make([]string, 0, len(a.Values))
		for _, item := range a.Values {
			v = append(v, item.String())
		}
	}
	if a.Elided {
		v = append(v, "...")
	}
	return strings.Join(v, ", ")
}

// equal returns true only if both arguments are exactly equal.
func (a *Args) equal(r *Args) bool {
	if a.Elided != r.Elided || len(a.Values) != len(r.Values) {
		return false
	}
	for i, l := range a.Values {
		if l != r.Values[i] {
			return false
		}
	}
	return true
}

// similar returns true if the two Args are equal or almost but not quite
// equal.
func (a *Args) similar(r *Args, similar Similarity) bool {
	if a.Elided != r.Elided || len(a.Values) != len(r.Values) {
		return false
	}
	if similar == AnyValue {
		return true
	}
	for i, l := range a.Values {
		switch similar {
		case ExactFlags, ExactLines:
			if l != r.Values[i] {
				return false
			}
		default:
			if l.IsPtr() != r.Values[i].IsPtr() || (!l.IsPtr() && l != r.Values[i]) {
				return false
			}
		}
	}
	return true
}

// merge merges two similar Args, zapping out differences.
func (a *Args) merge(r *Args) Args {
	out := Args{
		Values: make([]Arg, len(a.Values)),
		Elided: a.Elided,
	}
	for i, l := range a.Values {
		if l != r.Values[i] {
			out.Values[i].Name = "*"
			out.Values[i].Value = l.Value
		} else {
			out.Values[i] = l
		}
	}
	return out
}

// Call is an item in the stack trace.
type Call struct {
	SrcPath      string // Full path name of the source file as seen in the trace
	LocalSrcPath string // Full path name of the source file as seen in the host.
	Line         int    // Line number
	Func         Func   // Fully qualified function name (encoded).
	Args         Args   // Call arguments
	IsStdlib     bool   // true if it is a Go standard library function. This includes the 'go test' generated main executable.
}

// equal returns true only if both calls are exactly equal.
func (c *Call) equal(r *Call) bool {
	return c.SrcPath == r.SrcPath && c.Line == r.Line && c.Func == r.Func && c.Args.equal(&r.Args)
}

// similar returns true if the two Call are equal or almost but not quite
// equal.
func (c *Call) similar(r *Call, similar Similarity) bool {
	return c.SrcPath == r.SrcPath && c.Line == r.Line && c.Func == r.Func && c.Args.similar(&r.Args, similar)
}

// merge merges two similar Call, zapping out differences.
func (c *Call) merge(r *Call) Call {
	return Call{
		SrcPath:      c.SrcPath,
		Line:         c.Line,
		Func:         c.Func,
		Args:         c.Args.merge(&r.Args),
		LocalSrcPath: c.LocalSrcPath,
		IsStdlib:     c.IsStdlib,
	}
}

// SrcName returns the base file name of the source file.
func (c *Call) SrcName() string {
	return filepath.Base(c.SrcPath)
}

// SrcLine returns "source.go:line", including only the base file name.
func (c *Call) SrcLine() string {
	return fmt.Sprintf("%s:%d", c.SrcName(), c.Line)
}

// FullSrcLine returns "/path/to/source.go:line".
//
// This file path is mutated to look like the local path.
func (c *Call) FullSrcLine() string {
	return fmt.Sprintf("%s:%d", c.SrcPath, c.Line)
}

// PkgSrc is one directory plus the file name of the source file.
func (c *Call) PkgSrc() string {
	return filepath.Join(filepath.Base(filepath.Dir(c.SrcPath)), c.SrcName())
}

// IsPkgMain returns true if it is in the main package.
func (c *Call) IsPkgMain() bool {
	return c.Func.PkgName() == "main"
}

const testMainSrc = "_test" + string(os.PathSeparator) + "_testmain.go"

// updateLocations initializes LocalSrcPath and IsStdlib.
func (c *Call) updateLocations(goroot, localgoroot string, gopaths map[string]string) {
	if c.SrcPath != "" {
		// Always check GOROOT first, then GOPATH.
		if strings.HasPrefix(c.SrcPath, goroot) {
			// Replace remote GOROOT with local GOROOT.
			c.LocalSrcPath = filepath.Join(localgoroot, c.SrcPath[len(goroot):])
		} else {
			// Replace remote GOPATH with local GOPATH.
			c.LocalSrcPath = c.SrcPath
			// TODO(maruel): Sort for deterministic behavior?
			for prefix, dest := range gopaths {
				if strings.HasPrefix(c.SrcPath, prefix) {
					c.LocalSrcPath = filepath.Join(dest, c.SrcPath[len(prefix):])
					break
				}
			}
		}
	}
	// Consider _test/_testmain.go as stdlib since it's injected by "go test".
	c.IsStdlib = (goroot != "" && strings.HasPrefix(c.SrcPath, goroot)) || c.PkgSrc() == testMainSrc
}

// Stack is a call stack.
type Stack struct {
	Calls  []Call // Call stack. First is original function, last is leaf function.
	Elided bool   // Happens when there's >100 items in Stack, currently hardcoded in package runtime.
}

// equal returns true on if both call stacks are exactly equal.
func (s *Stack) equal(r *Stack) bool {
	if len(s.Calls) != len(r.Calls) || s.Elided != r.Elided {
		return false
	}
	for i := range s.Calls {
		if !s.Calls[i].equal(&r.Calls[i]) {
			return false
		}
	}
	return true
}

// similar returns true if the two Stack are equal or almost but not quite
// equal.
func (s *Stack) similar(r *Stack, similar Similarity) bool {
	if len(s.Calls) != len(r.Calls) || s.Elided != r.Elided {
		return false
	}
	for i := range s.Calls {
		if !s.Calls[i].similar(&r.Calls[i], similar) {
			return false
		}
	}
	return true
}

// merge merges two similar Stack, zapping out differences.
func (s *Stack) merge(r *Stack) *Stack {
	// Assumes similar stacks have the same length.
	out := &Stack{
		Calls:  make([]Call, len(s.Calls)),
		Elided: s.Elided,
	}
	for i := range s.Calls {
		out.Calls[i] = s.Calls[i].merge(&r.Calls[i])
	}
	return out
}

// less compares two Stack, where the ones that are less are more
// important, so they come up front.
//
// A Stack with more private functions is 'less' so it is at the top.
// Inversely, a Stack with only public functions is 'more' so it is at the
// bottom.
func (s *Stack) less(r *Stack) bool {
	lStdlib := 0
	lPrivate := 0
	for _, c := range s.Calls {
		if c.IsStdlib {
			lStdlib++
		} else {
			lPrivate++
		}
	}
	rStdlib := 0
	rPrivate := 0
	for _, s := range r.Calls {
		if s.IsStdlib {
			rStdlib++
		} else {
			rPrivate++
		}
	}
	if lPrivate > rPrivate {
		return true
	}
	if lPrivate < rPrivate {
		return false
	}
	if lStdlib > rStdlib {
		return false
	}
	if lStdlib < rStdlib {
		return true
	}

	// Stack lengths are the same.
	for x := range s.Calls {
		if s.Calls[x].Func.Raw < r.Calls[x].Func.Raw {
			return true
		}
		if s.Calls[x].Func.Raw > r.Calls[x].Func.Raw {
			return true
		}
		if s.Calls[x].PkgSrc() < r.Calls[x].PkgSrc() {
			return true
		}
		if s.Calls[x].PkgSrc() > r.Calls[x].PkgSrc() {
			return true
		}
		if s.Calls[x].Line < r.Calls[x].Line {
			return true
		}
		if s.Calls[x].Line > r.Calls[x].Line {
			return true
		}
	}
	return false
}

func (s *Stack) updateLocations(goroot, localgoroot string, gopaths map[string]string) {
	for i := range s.Calls {
		s.Calls[i].updateLocations(goroot, localgoroot, gopaths)
	}
}

// Signature represents the signature of one or multiple goroutines.
//
// It is effectively the stack trace plus the goroutine internal bits, like
// it's state, if it is thread locked, which call site created this goroutine,
// etc.
type Signature struct {
	// Use git grep 'gopark(|unlock)\(' to find them all plus everything listed
	// in runtime/traceback.go. Valid values includes:
	//     - chan send, chan receive, select
	//     - finalizer wait, mark wait (idle),
	//     - Concurrent GC wait, GC sweep wait, force gc (idle)
	//     - IO wait, panicwait
	//     - semacquire, semarelease
	//     - sleep, timer goroutine (idle)
	//     - trace reader (blocked)
	// Stuck cases:
	//     - chan send (nil chan), chan receive (nil chan), select (no cases)
	// Runnable states:
	//    - idle, runnable, running, syscall, waiting, dead, enqueue, copystack,
	// Scan states:
	//    - scan, scanrunnable, scanrunning, scansyscall, scanwaiting, scandead,
	//      scanenqueue
	State     string
	CreatedBy Call // Which other goroutine which created this one.
	SleepMin  int  // Wait time in minutes, if applicable.
	SleepMax  int  // Wait time in minutes, if applicable.
	Stack     Stack
	Locked    bool // Locked to an OS thread.
}

// equal returns true only if both signatures are exactly equal.
func (s *Signature) equal(r *Signature) bool {
	if s.State != r.State || !s.CreatedBy.equal(&r.CreatedBy) || s.Locked != r.Locked || s.SleepMin != r.SleepMin || s.SleepMax != r.SleepMax {
		return false
	}
	return s.Stack.equal(&r.Stack)
}

// similar returns true if the two Signature are equal or almost but not quite
// equal.
func (s *Signature) similar(r *Signature, similar Similarity) bool {
	if s.State != r.State || !s.CreatedBy.similar(&r.CreatedBy, similar) {
		return false
	}
	if similar == ExactFlags && s.Locked != r.Locked {
		return false
	}
	return s.Stack.similar(&r.Stack, similar)
}

// merge merges two similar Signature, zapping out differences.
func (s *Signature) merge(r *Signature) *Signature {
	min := s.SleepMin
	if r.SleepMin < min {
		min = r.SleepMin
	}
	max := s.SleepMax
	if r.SleepMax > max {
		max = r.SleepMax
	}
	return &Signature{
		State:     s.State,     // Drop right side.
		CreatedBy: s.CreatedBy, // Drop right side.
		SleepMin:  min,
		SleepMax:  max,
		Stack:     *s.Stack.merge(&r.Stack),
		Locked:    s.Locked || r.Locked, // TODO(maruel): This is weirdo.
	}
}

// less compares two Signature, where the ones that are less are more
// important, so they come up front. A Signature with more private functions is
// 'less' so it is at the top. Inversely, a Signature with only public
// functions is 'more' so it is at the bottom.
func (s *Signature) less(r *Signature) bool {
	if s.Stack.less(&r.Stack) {
		return true
	}
	if r.Stack.less(&s.Stack) {
		return false
	}
	if s.Locked && !r.Locked {
		return true
	}
	if r.Locked && !s.Locked {
		return false
	}
	if s.State < r.State {
		return true
	}
	if s.State > r.State {
		return false
	}
	return false
}

// SleepString returns a string "N-M minutes" if the goroutine(s) slept for a
// long time.
//
// Returns an empty string otherwise.
func (s *Signature) SleepString() string {
	if s.SleepMax == 0 {
		return ""
	}
	if s.SleepMin != s.SleepMax {
		return fmt.Sprintf("%d~%d minutes", s.SleepMin, s.SleepMax)
	}
	return fmt.Sprintf("%d minutes", s.SleepMax)
}

// CreatedByString return a short context about the origin of this goroutine
// signature.
func (s *Signature) CreatedByString(fullPath bool) string {
	created := s.CreatedBy.Func.PkgDotName()
	if created == "" {
		return ""
	}
	created += " @ "
	if fullPath {
		created += s.CreatedBy.FullSrcLine()
	} else {
		created += s.CreatedBy.SrcLine()
	}
	return created
}

func (s *Signature) updateLocations(goroot, localgoroot string, gopaths map[string]string) {
	s.CreatedBy.updateLocations(goroot, localgoroot, gopaths)
	s.Stack.updateLocations(goroot, localgoroot, gopaths)
}

// Goroutine represents the state of one goroutine, including the stack trace.
type Goroutine struct {
	Signature      // It's stack trace, internal bits, state, which call site created it, etc.
	ID        int  // Goroutine ID.
	First     bool // First is the goroutine first printed, normally the one that crashed.
}

// Private stuff.

// nameArguments is a post-processing step where Args are 'named' with numbers.
func nameArguments(goroutines []*Goroutine) {
	// Set a name for any pointer occurring more than once.
	type object struct {
		args      []*Arg
		inPrimary bool
		id        int
	}
	objects := map[uint64]object{}
	// Enumerate all the arguments.
	for i := range goroutines {
		for j := range goroutines[i].Stack.Calls {
			for k := range goroutines[i].Stack.Calls[j].Args.Values {
				arg := goroutines[i].Stack.Calls[j].Args.Values[k]
				if arg.IsPtr() {
					objects[arg.Value] = object{
						args:      append(objects[arg.Value].args, &goroutines[i].Stack.Calls[j].Args.Values[k]),
						inPrimary: objects[arg.Value].inPrimary || i == 0,
					}
				}
			}
		}
		// CreatedBy.Args is never set.
	}
	order := make(uint64Slice, 0, len(objects)/2)
	for k, obj := range objects {
		if len(obj.args) > 1 && obj.inPrimary {
			order = append(order, k)
		}
	}
	sort.Sort(order)
	nextID := 1
	for _, k := range order {
		for _, arg := range objects[k].args {
			arg.Name = fmt.Sprintf("#%d", nextID)
		}
		nextID++
	}

	// Now do the rest. This is done so the output is deterministic.
	order = make(uint64Slice, 0, len(objects))
	for k := range objects {
		order = append(order, k)
	}
	sort.Sort(order)
	for _, k := range order {
		// Process the remaining pointers, they were not referenced by primary
		// thread so will have higher IDs.
		if objects[k].inPrimary {
			continue
		}
		for _, arg := range objects[k].args {
			arg.Name = fmt.Sprintf("#%d", nextID)
		}
		nextID++
	}
}

type uint64Slice []uint64

func (a uint64Slice) Len() int           { return len(a) }
func (a uint64Slice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a uint64Slice) Less(i, j int) bool { return a[i] < a[j] }
