package cdn

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// TestRetryWorkerLookupRetriesATransientFailure pins that a lookup which failed without an answer
// is tried again. The blip that loses a whole job lasts less than a second; one more attempt is
// all it takes to ride over it.
func TestRetryWorkerLookupRetriesATransientFailure(t *testing.T) {
	name := sdk.RandomString(10)

	var calls int
	err := retryWorkerLookup(context.TODO(), name, func(ctx context.Context) error {
		calls++
		if calls < 3 {
			return fmt.Errorf("Transport Error: context deadline exceeded")
		}
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, calls)
}

// TestRetryWorkerLookupDoesNotRetryAMissingWorker pins the other half: a 404 is an answer, and
// asking again would only double the load on the API for a result that cannot change.
func TestRetryWorkerLookupDoesNotRetryAMissingWorker(t *testing.T) {
	name := sdk.RandomString(10)

	var calls int
	err := retryWorkerLookup(context.TODO(), name, func(ctx context.Context) error {
		calls++
		return sdk.NewError(sdk.ErrNotFound, fmt.Errorf("worker does not exist"))
	})
	require.Error(t, err)
	require.True(t, sdk.ErrorIs(err, sdk.ErrNotFound))
	require.Equal(t, 1, calls, "a worker that does not exist must be asked for once")
}

// TestRetryWorkerLookupCutsASlowAttempt is the reason the per attempt timeout exists. The client
// already retries, but its attempts share one context and it stops as soon as that context is
// done: one attempt slow enough to spend the whole budget leaves no room for the others, which is
// how a lookup that could have succeeded on a second try ends as a dropped log line.
func TestRetryWorkerLookupCutsASlowAttempt(t *testing.T) {
	name := sdk.RandomString(10)

	var calls int
	var deadlines []time.Duration
	start := time.Now()
	err := retryWorkerLookup(context.TODO(), name, func(ctx context.Context) error {
		calls++
		deadline, ok := ctx.Deadline()
		require.True(t, ok, "every attempt must be bounded on its own")
		deadlines = append(deadlines, time.Until(deadline).Round(100*time.Millisecond))
		<-ctx.Done() // an attempt that hangs until it is cut
		return ctx.Err()
	})
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Equal(t, workerLookupAttempts, calls, "a hanging attempt must not eat the attempts of the others")
	for _, d := range deadlines {
		require.Equal(t, workerLookupAttemptTimeout, d, "each attempt gets its own budget, not the remainder of the previous one")
	}
	require.Less(t, elapsed, workerLookupTimeout+time.Second, "the whole lookup stays bounded")
}

// TestRetryWorkerLookupRemembersAFailure pins the guard the sequential intake needs: messages of a
// connection are handled one at a time, so without it every line of a job whose worker cannot be
// looked up would pay the full budget in turn and stall the connection.
func TestRetryWorkerLookupRemembersAFailure(t *testing.T) {
	name := sdk.RandomString(10)
	t.Cleanup(func() { runCache.Delete(workerLookupFailureKey(name)) })

	var calls int
	fetch := func(ctx context.Context) error {
		calls++
		return fmt.Errorf("Transport Error: connection refused")
	}

	require.Error(t, retryWorkerLookup(context.TODO(), name, fetch))
	require.Equal(t, workerLookupAttempts, calls)

	start := time.Now()
	require.Error(t, retryWorkerLookup(context.TODO(), name, fetch))
	require.Equal(t, workerLookupAttempts, calls, "the next line must not pay the budget again")
	require.Less(t, time.Since(start), 100*time.Millisecond)
}
