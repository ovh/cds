package event_v2

import (
	"context"
	"sync"
	"time"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// stepUpdateWindow bounds how often the progress of a single job is announced. A job made of quick
// steps moves faster than a view can show, and every announcement costs a frame to each client
// watching that run, so the announcements falling in the same window are collapsed into one.
const stepUpdateWindow = time.Second

var runJobStepUpdates = &stepUpdates{pending: make(map[string]*heldStepUpdate)}

type heldStepUpdate struct {
	runJob sdk.V2WorkflowRunJob
	at     time.Time
}

// stepUpdates collapses the progress announcements of the jobs reporting to this API. An
// announcement carries the whole run job rather than what changed in it, so the last one of a window
// supersedes those it replaces and nothing is lost by holding the others back.
type stepUpdates struct {
	mutex sync.Mutex
	// A job is present as soon as its window is open; its value is what is waiting for the end of
	// that window, nil when nothing is.
	pending map[string]*heldStepUpdate
}

// take reports whether the progress of a job is to be announced right away, which it is when its
// window is closed. Otherwise it is held back until the window ends, taking the place of whatever
// was already waiting.
func (s *stepUpdates) take(rj sdk.V2WorkflowRunJob, at time.Time) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, windowIsOpen := s.pending[rj.ID]; windowIsOpen {
		s.pending[rj.ID] = &heldStepUpdate{runJob: rj, at: at}
		return false
	}

	s.pending[rj.ID] = nil
	return true
}

// flush ends the current window and returns what was held back during it. A job that held nothing
// back is forgotten, so that only the jobs that are moving are kept.
func (s *stepUpdates) flush() []heldStepUpdate {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	held := make([]heldStepUpdate, 0, len(s.pending))
	for id, update := range s.pending {
		if update == nil {
			delete(s.pending, id)
			continue
		}
		held = append(held, *update)
		s.pending[id] = nil
	}
	return held
}

// FlushRunJobStepUpdates announces the progress held back by the collapsing of the step updates. It
// runs until the context is done.
func FlushRunJobStepUpdates(ctx context.Context, store cache.Store) {
	tick := time.NewTicker(stepUpdateWindow)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			for _, update := range runJobStepUpdates.flush() {
				// Stamped with the moment the worker reported it, not the moment it is let through:
				// a view drops an event older than the last one it applied to the same job, which is
				// how an update held back past the end of its job is kept from overwriting it.
				publishRunJobEvent(ctx, store, sdk.EventRunJobStepUpdated, update.runJob, update.at)
			}
		}
	}
}
