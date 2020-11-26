// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sdk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"

	panicparsestack "github.com/maruel/panicparse/stack"
	"github.com/ovh/cds/sdk/log"

	"github.com/pkg/errors"
)

// GoRoutines contains list of routines that have to stay up
type GoRoutines struct {
	mutex  sync.Mutex
	status map[string]bool
}

// NewGoRoutines instanciates a new GoRoutineManager
func NewGoRoutines() *GoRoutines {
	return &GoRoutines{
		mutex:  sync.Mutex{},
		status: make(map[string]bool),
	}
}

// Run runs the function within a goroutine with a panic recovery, and keep GoRoutine status.
func (m *GoRoutines) Run(c context.Context, name string, fn func(ctx context.Context), writerFactories ...func(s string) (io.WriteCloser, error)) {
	m.mutex.Lock()
	m.status[name] = true
	m.mutex.Unlock()
	m.Exec(c, name, fn, writerFactories...)
}

// GetStatus returns the monitoring status of goroutines that should be running
func (m *GoRoutines) GetStatus() []MonitoringStatusLine {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	lines := make([]MonitoringStatusLine, len(m.status))
	i := 0
	for name, isActive := range m.status {
		status := MonitoringStatusAlert
		value := "NOT running"
		if isActive {
			status = MonitoringStatusOK
			value = "Running"
		}
		lines[i] = MonitoringStatusLine{
			Status:    status,
			Component: "goroutine/" + name,
			Value:     value,
		}
		i++
	}
	return lines
}

// Exec runs the function within a goroutine with a panic recovery
func (m *GoRoutines) Exec(c context.Context, name string, fn func(ctx context.Context), writerFactories ...func(s string) (io.WriteCloser, error)) {
	hostname, _ := os.Hostname()
	go func(ctx context.Context) {
		labels := pprof.Labels("goroutine-name", name, "goroutine-hostname", hostname, "goroutine-id", fmt.Sprintf("%d", GoroutineID()))
		goroutineCtx := pprof.WithLabels(ctx, labels)
		pprof.SetGoroutineLabels(goroutineCtx)

		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 1<<16)
				runtime.Stack(buf, false)
				uuid := UUID()
				log.ErrorWithFields(ctx, log.Fields{"stack_trace": string(buf)}, "[PANIC][%s] %s failed (%s)", hostname, name, uuid)

				for _, f := range writerFactories {
					w, err := f(uuid)
					if err != nil {
						log.Error(ctx, "unable open writer %s ¯\\_(ツ)_/¯ (%v)", uuid, err)
						continue
					}
					if _, err := io.Copy(w, bytes.NewReader(buf)); err != nil {
						log.Error(ctx, "unable to write %s ¯\\_(ツ)_/¯ (%v)", uuid, err)
						continue
					}
					if err := w.Close(); err != nil {
						log.Error(ctx, "unable to close %s ¯\\_(ツ)_/¯ (%v)", uuid, err)
					}
				}
			}
			m.mutex.Lock()
			if _, ok := m.status[name]; ok {
				m.status[name] = false
			}
			m.mutex.Unlock()
		}()

		fn(goroutineCtx)
	}(c)
}

// code from https://github.com/golang/net/blob/master/http2/gotrack.go

var goroutineSpace = []byte("goroutine ")

var littleBuf = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 64)
		return &buf
	},
}

func GoroutineID() uint64 {
	bp := littleBuf.Get().(*[]byte)
	defer littleBuf.Put(bp)
	b := *bp
	b = b[:runtime.Stack(b, false)]
	// Parse the 4707 out of "goroutine 4707 ["
	b = bytes.TrimPrefix(b, goroutineSpace)
	i := bytes.IndexByte(b, ' ')
	if i < 0 {
		panic(fmt.Sprintf("No space found in %q", b))
	}
	b = b[:i]
	n, err := parseUintBytes(b, 10, 64)
	if err != nil {
		panic(fmt.Sprintf("Failed to parse goroutine ID out of %q: %v", b, err))
	}
	return n
}

func ListGoroutines() ([]*panicparsestack.Goroutine, error) {
	var w = new(bytes.Buffer)
	if err := writeGoroutineStacks(w); err != nil {
		return nil, err
	}
	all, err := parseGoRoutineStacks(w, nil)
	if err != nil {
		return nil, err
	}
	return all, nil
}

func parseUintBytes(s []byte, base int, bitSize int) (n uint64, err error) {
	var cutoff, maxVal uint64

	if bitSize == 0 {
		bitSize = int(strconv.IntSize)
	}

	s0 := s
	switch {
	case len(s) < 1:
		err = strconv.ErrSyntax
		goto Error

	case 2 <= base && base <= 36:
		// valid base; nothing to do

	case base == 0:
		// Look for octal, hex prefix.
		switch {
		case s[0] == '0' && len(s) > 1 && (s[1] == 'x' || s[1] == 'X'):
			base = 16
			s = s[2:]
			if len(s) < 1 {
				err = strconv.ErrSyntax
				goto Error
			}
		case s[0] == '0':
			base = 8
		default:
			base = 10
		}

	default:
		err = errors.New("invalid base " + strconv.Itoa(base))
		goto Error
	}

	n = 0
	cutoff = cutoff64(base)
	maxVal = 1<<uint(bitSize) - 1

	for i := 0; i < len(s); i++ {
		var v byte
		d := s[i]
		switch {
		case '0' <= d && d <= '9':
			v = d - '0'
		case 'a' <= d && d <= 'z':
			v = d - 'a' + 10
		case 'A' <= d && d <= 'Z':
			v = d - 'A' + 10
		default:
			n = 0
			err = strconv.ErrSyntax
			goto Error
		}
		if int(v) >= base {
			n = 0
			err = strconv.ErrSyntax
			goto Error
		}

		if n >= cutoff {
			// n*base overflows
			n = 1<<64 - 1
			err = strconv.ErrRange
			goto Error
		}
		n *= uint64(base)

		n1 := n + uint64(v)
		if n1 < n || n1 > maxVal {
			// n+v overflows
			n = 1<<64 - 1
			err = strconv.ErrRange
			goto Error
		}
		n = n1
	}

	return n, nil

Error:
	return n, &strconv.NumError{Func: "ParseUint", Num: string(s0), Err: err}
}

// Return the first number n such that n*base >= 1<<64.
func cutoff64(base int) uint64 {
	if base < 2 {
		return 0
	}
	return (1<<64-1)/uint64(base) + 1
}

func writeGoroutineStacks(w io.Writer) error {
	buf := make([]byte, 1<<20)
	for i := 0; ; i++ {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			buf = buf[:n]
			break
		}
		if len(buf) >= 256<<20 {
			// Filled 256 MB - stop there.
			break
		}
		buf = make([]byte, 2*len(buf))
	}
	_, err := w.Write(buf)
	return WithStack(err)
}

func parseGoRoutineStacks(r io.Reader, w io.Writer) ([]*panicparsestack.Goroutine, error) {
	if w == nil {
		w = ioutil.Discard
	}
	goroutines, err := panicparsestack.ParseDump(r, w, true)
	if err != nil {
		return nil, WithStack(err)
	}
	return goroutines.Goroutines, nil
}
