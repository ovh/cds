package sdk

import (
	"bytes"
	"context"
	"fmt"
	"runtime/pprof"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_GoroutineTools(t *testing.T) {
	t.Run("GoroutineID()", func(t *testing.T) {
		require.NotEqual(t, uint64(0), GoroutineID())
	})

	t.Run("GoRoutineStacks(...)", func(t *testing.T) {
		var w = new(bytes.Buffer)
		require.NoError(t, writeGoroutineStacks(w))
		_, err := parseGoRoutineStacks(w, nil)
		require.NoError(t, err)
	})

	t.Run("GoRoutineRun", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
		t.Cleanup(cancel)
		m := NewGoRoutines(ctx)

		ctxRoutine, routineCancel := context.WithCancel(context.TODO())
		m.Run(context.TODO(), "test_goroutine_run", func(_ context.Context) {
			<-ctxRoutine.Done()
		})

		s := m.GoRoutine("test_goroutine_run")
		require.NotNil(t, s)
		require.True(t, s.Active)
		require.Len(t, m.GetStatus(), 1)

		routineCancel()

		// Wait for the routine status to change
		ticker := time.NewTicker(1 * time.Millisecond)
		t.Cleanup(ticker.Stop)
	wait:
		for {
			select {
			case <-ctx.Done():
				break wait
			case <-ticker.C:
				s = m.GoRoutine("test_goroutine_run")
				if s != nil && !s.Active {
					break wait
				}
			}
		}

		s = m.GoRoutine("test_goroutine_run")
		require.NotNil(t, s)
		require.False(t, s.Active)
	})

	t.Run("GoRoutineRunCancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		t.Cleanup(cancel)
		m := NewGoRoutines(ctx)

		ctxToCancelled, cancelRoutine := context.WithTimeout(context.TODO(), 5*time.Second)
		var cancelled bool
		m.Run(context.TODO(), "test_goroutine_run_cancel", func(ctx context.Context) {
			<-ctx.Done()
			cancelled = true
			cancelRoutine()
		})

		require.False(t, cancelled)
		m.Stop("test_goroutine_run_cancel")
		<-ctxToCancelled.Done()
		require.True(t, cancelled)
	})

	// Each manager carries a monitoring goroutine that only ends with its context, so building one
	// per unit of work leaks one goroutine per call. Callers share the manager of their service.
	t.Run("NewGoRoutines costs one goroutine until its context ends", func(t *testing.T) {
		// Count only the goroutines started from this context, through a pprof label they inherit:
		// the process-wide count also moves with the goroutines of other tests, which made it flaky
		id := RandomString(10)
		ctx, cancel := context.WithCancel(pprof.WithLabels(context.TODO(), pprof.Labels(testGoroutinesLabel, id)))
		t.Cleanup(cancel)
		for i := 0; i < 50; i++ {
			_ = NewGoRoutines(ctx)
		}

		require.Eventually(t, func() bool {
			return countLabelledGoroutines(t, testGoroutinesLabel, id) == 50
		}, 5*time.Second, 10*time.Millisecond, "expected one monitoring goroutine per manager")

		cancel()

		require.Eventually(t, func() bool {
			return countLabelledGoroutines(t, testGoroutinesLabel, id) == 0
		}, 15*time.Second, 50*time.Millisecond, "monitoring goroutines outlived their context")
	})

	t.Run("GoRoutineRunWithRestart", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 15*time.Second)
		t.Cleanup(cancel)
		m := NewGoRoutines(ctx)

		var count int
		m.RunWithRestart(context.TODO(), "test_goroutine_run_with_restart", func(ctx context.Context) {
			count++
		})

		// the routine should have restart 1 time
		<-ctx.Done()
		require.Equal(t, 2, count)
	})
}

const testGoroutinesLabel = "test-goroutines"

// countLabelledGoroutines counts the running goroutines whose pprof labels hold key=value
func countLabelledGoroutines(t *testing.T, key, value string) int {
	var buf bytes.Buffer
	require.NoError(t, pprof.Lookup("goroutine").WriteTo(&buf, 1))

	label := fmt.Sprintf("%q:%q", key, value)
	var count, groupSize int
	for _, line := range strings.Split(buf.String(), "\n") {
		// A group of identical goroutines starts with "<count> @ <pcs>", then its labels if any
		if fields := strings.Fields(line); len(fields) > 1 && fields[1] == "@" {
			n, err := strconv.Atoi(fields[0])
			require.NoError(t, err)
			groupSize = n
			continue
		}
		if strings.HasPrefix(line, "# labels: ") && strings.Contains(line, label) {
			count += groupSize
		}
	}
	return count
}
