package hooks

import (
	"context"
	"time"

	"github.com/ovh/cds/sdk/log"
)

func (s *Service) runScheduler(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	go func() {
		if err := s.dequeueLongRunningTasks(ctx); err != nil {
			log.Error("Hooks> runScheduler> dequeueLongRunningTasks> %v", err)
			cancel()
		}
	}()

	//TODO scheduler
	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) dequeueLongRunningTasks(c context.Context) error {
	for {
		if c.Err() != nil {
			return c.Err()
		}

		var taskKey string
		s.Cache.DequeueWithContext(c, schedulerQueueLongRuningTasksKey, &taskKey)

		var t = LongRunningTaskExecution{}
		if !s.Cache.Get(taskKey, &t) {
			continue
		}
		t.ProcessingTimestamp = time.Now().UnixNano()
		s.Dao.SaveLongRunningTaskExecution(&t)

		if err := s.doLongRunningTask(c, &t); err != nil {
			log.Error("Hooks> doLongRunningTask failed: %v", err)
			t.LastError = err.Error()
		}

		s.Dao.SaveLongRunningTaskExecution(&t)
		continue
	}
}
