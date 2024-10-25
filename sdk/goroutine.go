// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sdk

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"

	"github.com/maruel/panicparse/v2/stack"
	"github.com/pkg/errors"
	"github.com/rockbears/log"

	cdslog "github.com/ovh/cds/sdk/log"
)

type GoRoutine struct {
	ctx     context.Context
	cancel  func()
	Name    string
	Func    func(ctx context.Context)
	Restart bool
	Active  bool
	mutex   sync.RWMutex
}

// GoRoutines contains list of routines that have to stay up
type GoRoutines struct {
	mutex  sync.RWMutex
	status []*GoRoutine
}

func (m *GoRoutines) GoRoutine(name string) *GoRoutine {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, g := range m.status {
		if g.Name == name {
			return g
		}
	}
	return nil
}

func (m *GoRoutines) Stop(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for i, g := range m.status {
		if g.Name == name {
			if g.cancel != nil {
				g.cancel()
			}
			m.status = append(m.status[:i], m.status[i+1:]...)
			break
		}
	}
}

// NewGoRoutines instanciates a new GoRoutineManager
func NewGoRoutines(ctx context.Context) *GoRoutines {
	m := &GoRoutines{}
	m.Exec(ctx, "GoRoutines-restart", func(ctx context.Context) {
		m.restartGoRoutines(ctx)
	})
	return m
}

func (m *GoRoutines) restartGoRoutines(ctx context.Context) {
	t := time.NewTicker(10 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			for _, g := range m.status {
				if g.cancel != nil {
					g.cancel()
				}
			}
			m.status = nil
			return
		case <-t.C:
			m.runRestartGoRoutines(ctx)
		}
	}
}

func (m *GoRoutines) runRestartGoRoutines(ctx context.Context) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, g := range m.status {
		g.mutex.RLock()
		active := g.Active
		g.mutex.RUnlock()
		if !active && g.Restart {
			log.Info(ctx, "restarting goroutine %q", g.Name)
			m.exec(g)
		}
	}
}

// Run runs the function within a goroutine with a panic recovery, and keep GoRoutine status.
func (m *GoRoutines) Run(c context.Context, name string, fn func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(c)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	g := &GoRoutine{
		ctx:     ctx,
		cancel:  cancel,
		Name:    name,
		Func:    fn,
		Active:  true,
		Restart: false,
	}
	m.status = append(m.status, g)
	m.exec(g)
}

// RunWithRestart runs the function within a goroutine with a panic recovery, and keep GoRoutine status.
// if the goroutine is stopped, it will ne restarted
func (m *GoRoutines) RunWithRestart(c context.Context, name string, fn func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(c)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	g := &GoRoutine{
		ctx:     ctx,
		cancel:  cancel,
		Name:    name,
		Func:    fn,
		Active:  true,
		Restart: true,
	}
	m.status = append(m.status, g)
	m.exec(g)
}

// GetStatus returns the monitoring status of goroutines that should be running
func (m *GoRoutines) GetStatus() []MonitoringStatusLine {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	lines := make([]MonitoringStatusLine, len(m.status))
	i := 0
	for _, g := range m.status {
		status := MonitoringStatusAlert
		value := "NOT running"
		g.mutex.RLock()
		if g.Active {
			status = MonitoringStatusOK
			value = "Running"
		}
		g.mutex.RUnlock()
		lines[i] = MonitoringStatusLine{
			Status:    status,
			Component: "goroutine/" + g.Name,
			Value:     value,
		}
		i++
	}
	return lines
}

func (m *GoRoutines) exec(g *GoRoutine) {
	hostname, _ := os.Hostname()

	go func(ctx context.Context) {
		ctx = context.WithValue(ctx, cdslog.Goroutine, g.Name)

		labels := pprof.Labels(
			"goroutine-name", g.Name,
			"goroutine-hostname", hostname,
			"goroutine-id", fmt.Sprintf("%d", GoroutineID()),
		)
		goroutineCtx := pprof.WithLabels(ctx, labels)
		pprof.SetGoroutineLabels(goroutineCtx)

		defer func() {
			if r := recover(); r != nil {
				buf := make([]byte, 1<<16)
				buf = buf[:runtime.Stack(buf, false)]
				ctx = context.WithValue(ctx, cdslog.Stacktrace, string(buf))
				log.Error(ctx, "[PANIC][%s] %s failed", hostname, g.Name)
			}
			g.mutex.Lock()
			g.Active = false
			g.mutex.Unlock()
		}()

		g.mutex.Lock()
		g.Active = true
		g.mutex.Unlock()
		g.Func(goroutineCtx)
	}(g.ctx)
}

// Exec runs the function within a goroutine with a panic recovery
func (m *GoRoutines) Exec(c context.Context, name string, fn func(ctx context.Context)) {
	g := &GoRoutine{
		ctx:     c,
		Name:    name,
		Func:    fn,
		Active:  true,
		Restart: false,
	}
	m.exec(g)
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

func ListGoroutines() ([]*stack.Goroutine, error) {
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

func parseGoRoutineStacks(r io.Reader, w io.Writer) ([]*stack.Goroutine, error) {
	if w == nil {
		w = io.Discard
	}
	s, _, err := stack.ScanSnapshot(r, w, stack.DefaultOpts())
	if err != nil && err != io.EOF {
		return nil, WithStack(err)
	}
	return s.Goroutines, nil
}
