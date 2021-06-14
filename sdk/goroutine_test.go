package sdk

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GoroutineTools(t *testing.T) {
	t.Run("GoroutineID()", func(t *testing.T) {
		id := GoroutineID()
		var zero uint64
		assert.NotEqual(t, zero, id)
	})

	t.Run("writeGoroutineStacks(...)", func(t *testing.T) {
		ctx := context.Background()
		var wg = new(sync.WaitGroup)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		NewGoRoutines(ctx).Exec(ctx, "test_goroutine", func(ctx context.Context) {
			wg.Add(1)
			<-ctx.Done()
			wg.Done()
		})

		var w = new(bytes.Buffer)
		err := writeGoroutineStacks(w)
		assert.NoError(t, err)
		t.Log(w.String())
		wg.Wait()
	})

	t.Run("parseGoRoutineStacks(...)", func(t *testing.T) {
		ctx := context.Background()
		var wg = new(sync.WaitGroup)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		NewGoRoutines(ctx).Exec(ctx, "test_goroutine", func(ctx context.Context) {
			wg.Add(1)
			<-ctx.Done()
			wg.Done()
		})

		var w = new(bytes.Buffer)
		err := writeGoroutineStacks(w)
		assert.NoError(t, err)

		_, err = parseGoRoutineStacks(w, nil)
		assert.NoError(t, err)
		wg.Wait()
	})

	t.Run("GoRoutineLoop", func(t *testing.T) {
		ctx := context.Background()
		var wg = new(sync.WaitGroup)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		m := NewGoRoutines(ctx)
		m.Run(ctx, "test_goroutine_loop", func(ctx context.Context) {
			wg.Add(1)
			s := m.GoRoutine("test_goroutine_loop")
			require.NotNil(t, s)
			require.True(t, s.Active)
			<-ctx.Done()
			wg.Done()
		})

		s := m.GoRoutine("test_goroutine_loop")
		require.NotNil(t, s)
		require.Equal(t, 1, len(m.GetStatus()))
	})
}
