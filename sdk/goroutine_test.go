package sdk

import (
	"bytes"
	"context"
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
