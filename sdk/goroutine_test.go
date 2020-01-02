package sdk

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_GoroutineTools(t *testing.T) {
	t.Run("GoroutineID()", func(t *testing.T) {
		id := GoroutineID()
		var zero uint64
		assert.NotEqual(t, zero, id)
	})

	t.Run("WriteGoroutineStacks(...)", func(t *testing.T) {
		ctx := context.Background()
		var wg = new(sync.WaitGroup)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		GoRoutine(ctx, "test_goroutine", func(ctx context.Context) {
			wg.Add(1)
			<-ctx.Done()
			wg.Done()
		})

		var w = new(bytes.Buffer)
		err := WriteGoroutineStacks(w)
		assert.NoError(t, err)
		t.Log(w.String())
		wg.Wait()
	})

	t.Run("ParseGoRoutineStacks(...)", func(t *testing.T) {
		ctx := context.Background()
		var wg = new(sync.WaitGroup)
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		GoRoutine(ctx, "test_goroutine", func(ctx context.Context) {
			wg.Add(1)
			<-ctx.Done()
			wg.Done()
		})

		var w = new(bytes.Buffer)
		err := WriteGoroutineStacks(w)
		assert.NoError(t, err)

		_, err = ParseGoRoutineStacks(w, nil)
		assert.NoError(t, err)
		wg.Wait()
	})
}
