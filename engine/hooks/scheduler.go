package hooks

import (
	"context"
	"sort"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// Entry point of the internal scheduler
func (s *Service) runScheduler(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer cancel()

	go func() {
		if err := s.dequeueTaskExecutions(ctx); err != nil {
			log.Error("Hooks> runScheduler> dequeueLongRunningTasks> %v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.retryTaskExecutionsRoutine(ctx); err != nil {
			log.Error("Hooks> runScheduler> retryTaskExecutionsRoutine> %v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.enqueueScheduledTaskExecutionsRoutine(ctx); err != nil {
			log.Error("Hooks> runScheduler> retryTaskExecutionsRoutine> %v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.deleteTaskExecutionsRoutine(ctx); err != nil {
			log.Error("Hooks> runScheduler> deleteTaskExecutionsRoutine> %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}

// Every x seconds, the scheduler try to relaunch all tasks which have never been processed, or in error
func (s *Service) retryTaskExecutionsRoutine(c context.Context) error {
	tick := time.NewTicker(time.Duration(s.Cfg.RetryDelay) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			return c.Err()
		case <-tick.C:
			tasks, err := s.Dao.FindAllTasks()
			if err != nil {
				log.Error("Hooks> retryTaskExecutionsRoutine > Unable to find all tasks: %v", err)
				continue
			}
			for _, t := range tasks {
				execs, err := s.Dao.FindAllTaskExecutions(&t)
				if err != nil {
					log.Error("Hooks> retryTaskExecutionsRoutine > Unable to find all task executions (%s): %v", t.UUID, err)
					continue
				}
				for _, e := range execs {
					if e.Status == TaskExecutionDoing || e.Status == TaskExecutionScheduled {
						continue
					}
					// old hooks
					if e.ProcessingTimestamp == 0 && e.Timestamp < time.Now().Add(-2*time.Minute).UnixNano() {
						log.Warning("Enqueing very old hooks %s %d/%d type:%s status:%s err:%s", e.UUID, e.NbErrors, s.Cfg.RetryError, e.Type, e.Status, e.LastError)
						s.Dao.EnqueueTaskExecution(&e)
					}
					if e.NbErrors < s.Cfg.RetryError && e.LastError != "" {
						log.Warning("Enqueing with lastError %s %d/%d type:%s status:%s len:%d err:%s", e.UUID, e.NbErrors, s.Cfg.RetryError, e.Type, e.Status, len(e.LastError), e.LastError)
						s.Dao.EnqueueTaskExecution(&e)
						continue
					}
				}
			}
		}
	}
}

// Every 30 seconds, the scheduler try to launch all scheduled tasks (scheduler or repoPoller) which have never been processed
func (s *Service) enqueueScheduledTaskExecutionsRoutine(c context.Context) error {
	tick := time.NewTicker(time.Duration(30) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			return c.Err()
		case <-tick.C:
			tasks, err := s.Dao.FindAllTasks()
			if err != nil {
				log.Error("Hooks> enqueueScheduledTaskExecutionsRoutine > Unable to find all tasks: %v", err)
				continue
			}
			for _, t := range tasks {
				execs, err := s.Dao.FindAllTaskExecutions(&t)
				if err != nil {
					log.Error("Hooks> enqueueScheduledTaskExecutionsRoutine > Unable to find all task executions (%s): %v", t.UUID, err)
					continue
				}
				for _, e := range execs {
					if e.Status == TaskExecutionScheduled && e.ProcessingTimestamp == 0 && e.Timestamp <= time.Now().UnixNano() {
						// update status before enqueue
						// this will avoid to re-enqueue the same scheduled task execution if the dequeue take more than 30s (ticker of this goroutine)
						e.Status = ""
						s.Dao.SaveTaskExecution(&e)
						log.Info("Enqueing scheduler or repoPoller task %s type:%s", e.UUID, e.Type)
						s.Dao.EnqueueTaskExecution(&e)
					}
				}
			}
		}
	}
}

// Every 60 seconds, old executions of each task are deleted
func (s *Service) deleteTaskExecutionsRoutine(c context.Context) error {
	tick := time.NewTicker(time.Duration(60) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-c.Done():
			return c.Err()
		case <-tick.C:
			tasks, err := s.Dao.FindAllTasks()
			if err != nil {
				log.Error("Hooks> deleteTaskExecutionsRoutine > Unable to find all tasks: %v", err)
				continue
			}
			for _, t := range tasks {
				execs, err := s.Dao.FindAllTaskExecutions(&t)
				if err != nil {
					log.Error("Hooks> deleteTaskExecutionsRoutine > Unable to find all task executions (%s): %v", t.UUID, err)
					continue
				}
				sort.Slice(execs, func(i, j int) bool {
					return execs[i].Timestamp > execs[j].Timestamp
				})

				for i, e := range execs {
					if i >= s.Cfg.ExecutionHistory && e.ProcessingTimestamp != 0 {
						s.Dao.DeleteTaskExecution(&e)
					}
				}
			}
		}
	}
}

// Get from queue task execution
func (s *Service) dequeueTaskExecutions(c context.Context) error {
	for {
		if c.Err() != nil {
			return c.Err()
		}

		// Dequeuing context
		var taskKey string
		s.Cache.DequeueWithContext(c, schedulerQueueKey, &taskKey)

		// Load the task execution
		var t = sdk.TaskExecution{}
		if !s.Cache.Get(taskKey, &t) {
			continue
		}
		t.ProcessingTimestamp = time.Now().UnixNano()
		t.LastError = ""
		t.Status = TaskExecutionDoing
		s.Dao.SaveTaskExecution(&t)

		var restartTask bool
		var saveTaskExecution bool

		task := s.Dao.FindTask(t.UUID)
		if task == nil {
			log.Error("Hooks> dequeueTaskExecutions failed: Task %s not found", t.UUID)
			t.LastError = "Internal Error: Task not found"
			t.NbErrors++
			log.Info("Hooks> Deleting task execution %s", t.UUID)
			s.Dao.DeleteTaskExecution(&t)

		} else if task.Stopped {
			t.LastError = "Executions skipped: Task has been stopped"
			t.NbErrors++
			saveTaskExecution = true
		} else {
			restartTask = true
			saveTaskExecution = true
			if err := s.doTask(c, task, &t); err != nil {
				log.Error("Hooks> dequeueTaskExecutions %s failed err[%d]: %v", t.UUID, t.NbErrors, err)
				t.LastError = err.Error()
				t.NbErrors++
				saveTaskExecution = true
			}
		}

		//Save the execution
		if saveTaskExecution {
			t.Status = TaskExecutionDone
			t.ProcessingTimestamp = time.Now().UnixNano()
			s.Dao.SaveTaskExecution(&t)
		}

		//Start (or restart) the task
		if restartTask {
			s.startTask(c, task)
		}
	}
}
