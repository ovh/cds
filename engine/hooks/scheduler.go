package hooks

import (
	"context"
	"sort"
	"strings"
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
			log.Error(ctx, "runScheduler> dequeueLongRunningTasks> %v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.retryTaskExecutionsRoutine(ctx); err != nil {
			log.Error(ctx, "runScheduler> retryTaskExecutionsRoutine> %v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.enqueueScheduledTaskExecutionsRoutine(ctx); err != nil {
			log.Error(ctx, "runScheduler> enqueueScheduledTaskExecutionsRoutine> %v", err)
			cancel()
		}
	}()

	go func() {
		if err := s.deleteTaskExecutionsRoutine(ctx); err != nil {
			log.Error(ctx, "runScheduler> deleteTaskExecutionsRoutine> %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	return ctx.Err()
}

// Every x seconds, the scheduler try to relaunch all tasks which have never been processed, or in error
func (s *Service) retryTaskExecutionsRoutine(ctx context.Context) error {
	tick := time.NewTicker(time.Duration(s.Cfg.RetryDelay) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			size, err := s.Dao.QueueLen()
			if err != nil {
				log.Error(ctx, "retryTaskExecutionsRoutine > Unable to get queueLen: %v", err)
				continue
			}
			if size > 20 {
				log.Warning(ctx, "too many tasks in scheduler for now, skipped this retry ticker. size:%d", size)
				continue
			}

			if s.Maintenance {
				log.Info(ctx, "retryTaskExecutionsRoutine> Maintenance enable, wait 1 minute. Queue %d", size)
				time.Sleep(1 * time.Minute)
				continue
			}

			tasks, err := s.Dao.FindAllTasks(ctx)
			if err != nil {
				log.Error(ctx, "retryTaskExecutionsRoutine > Unable to find all tasks: %v", err)
				continue
			}
			for _, t := range tasks {
				execs, err := s.Dao.FindAllTaskExecutions(ctx, &t)
				if err != nil {
					log.Error(ctx, "retryTaskExecutionsRoutine > Unable to find all task executions (%s): %v", t.UUID, err)
					continue
				}
				for _, e := range execs {
					if e.Status == TaskExecutionDoing || e.Status == TaskExecutionScheduled || e.Status == TaskExecutionEnqueued {
						continue
					}

					// old hooks
					if e.ProcessingTimestamp == 0 && e.Timestamp < time.Now().Add(-2*time.Minute).UnixNano() {
						if e.UUID == "" {
							log.Warning(ctx, "retryTaskExecutionsRoutine > Very old hook without UUID %d/%d type:%s status:%s timestamp:%d err:%v", e.NbErrors, s.Cfg.RetryError, e.Type, e.Status, e.Timestamp, e.LastError)
							continue
						}
						e.Status = TaskExecutionEnqueued
						if err := s.Dao.SaveTaskExecution(&e); err != nil {
							log.Warning(ctx, "retryTaskExecutionsRoutine> unable to save task execution for old hook %s: %v", e.UUID, err)
							continue
						}
						log.Warning(ctx, "retryTaskExecutionsRoutine > Enqueing very old hooks %s %d/%d type:%s status:%s timestamp:%d err:%v", e.UUID, e.NbErrors, s.Cfg.RetryError, e.Type, e.Status, e.Timestamp, e.LastError)
						if err := s.Dao.EnqueueTaskExecution(ctx, &e); err != nil {
							log.Error(ctx, "retryTaskExecutionsRoutine > error on EnqueueTaskExecution: %v", err)
						}
					}
					if e.NbErrors < s.Cfg.RetryError && e.LastError != "" {
						// avoid re-enqueue if the lastError is about a git branch not found
						// the branch was deleted from git repository, it will never work
						if strings.Contains(e.LastError, "branchName parameter must be provided") {
							log.Warning(ctx, "retryTaskExecutionsRoutine > Do not re-enqueue this taskExecution with lastError %s %d/%d type:%s status:%s len:%d err:%s", e.UUID, e.NbErrors, s.Cfg.RetryError, e.Type, e.Status, len(e.LastError), e.LastError)
							if err := s.Dao.DeleteTaskExecution(&e); err != nil {
								log.Error(ctx, "retryTaskExecutionsRoutine > error on DeleteTaskExecution: %v", err)
							}
							continue
						}
						e.Status = TaskExecutionEnqueued
						if err := s.Dao.SaveTaskExecution(&e); err != nil {
							log.Warning(ctx, "retryTaskExecutionsRoutine> unable to save task execution for %s: %v", e.UUID, err)
							continue
						}
						log.Warning(ctx, "retryTaskExecutionsRoutine > Enqueing with lastError %s %d/%d type:%s status:%s len:%d err:%s", e.UUID, e.NbErrors, s.Cfg.RetryError, e.Type, e.Status, len(e.LastError), e.LastError)
						if err := s.Dao.EnqueueTaskExecution(ctx, &e); err != nil {
							log.Error(ctx, "retryTaskExecutionsRoutine > error on EnqueueTaskExecution: %v", err)
						}
						continue
					}
				}
			}
		}
	}
}

// Every 10 seconds, the scheduler try to launch all scheduled tasks which have never been processed
func (s *Service) enqueueScheduledTaskExecutionsRoutine(ctx context.Context) error {
	tick := time.NewTicker(time.Duration(10) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Error(ctx, "enqueueScheduledTaskExecutionsRoutine > exiting goroutine: %v", ctx.Err())
			return ctx.Err()
		case <-tick.C:
			tasks, err := s.Dao.FindAllTasks(ctx)
			if err != nil {
				log.Error(ctx, "enqueueScheduledTaskExecutionsRoutine > Unable to find all tasks: %v", err)
				continue
			}
			for _, t := range tasks {
				execs, err := s.Dao.FindAllTaskExecutions(ctx, &t)
				if err != nil {
					log.Error(ctx, "enqueueScheduledTaskExecutionsRoutine > Unable to find all task executions (%s): %v", t.UUID, err)
					continue
				}
				alreadyEnqueued := false
				for _, e := range execs {
					if e.Status == TaskExecutionScheduled && e.ProcessingTimestamp == 0 && e.Timestamp <= time.Now().UnixNano() {
						// update status before enqueue
						// this will avoid to re-enqueue the same scheduled task execution if the dequeue take more than 30s (ticker of this goroutine)
						if alreadyEnqueued {
							log.Info(ctx, "enqueueScheduledTaskExecutionsRoutine > task execution already enqueued for this task %s of type %s- delete it", e.UUID, e.Type)
							if err := s.Dao.DeleteTaskExecution(&e); err != nil {
								log.Error(ctx, "enqueueScheduledTaskExecutionsRoutine > error on DeleteTaskExecution: %v", err)
							}
						} else {
							e.Status = TaskExecutionEnqueued
							s.Dao.SaveTaskExecution(&e)
							log.Info(ctx, "enqueueScheduledTaskExecutionsRoutine > Enqueing %s task %s:%d", e.Type, e.UUID, e.Timestamp)
							if err := s.Dao.EnqueueTaskExecution(ctx, &e); err != nil {
								log.Error(ctx, "enqueueScheduledTaskExecutionsRoutine > error on EnqueueTaskExecution: %v", err)
							}
							// this will avoid to re-enqueue the same scheduled task execution if the dequeue take more than 30s (ticker of this goroutine)
							if e.Type == TypeRepoPoller || e.Type == TypeScheduler {
								alreadyEnqueued = true
							}

						}

					}
				}
			}
		}
	}
}

// Every 60 seconds, old executions of each task are deleted
func (s *Service) deleteTaskExecutionsRoutine(ctx context.Context) error {
	tick := time.NewTicker(time.Duration(60) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			tasks, err := s.Dao.FindAllTasks(ctx)
			if err != nil {
				log.Error(ctx, "deleteTaskExecutionsRoutine > Unable to find all tasks: %v", err)
				continue
			}
			for _, t := range tasks {
				taskToDelete := false
				execs, err := s.Dao.FindAllTaskExecutions(ctx, &t)
				if err != nil {
					log.Error(ctx, "deleteTaskExecutionsRoutine > Unable to find all task executions (%s): %v", t.UUID, err)
					continue
				}
				sort.Slice(execs, func(i, j int) bool {
					return execs[i].Timestamp > execs[j].Timestamp
				})

				for i, e := range execs {
					switch e.Type {
					// Delete all branch deletion task execution
					case TypeBranchDeletion:
						if e.Status == TaskExecutionDone && e.ProcessingTimestamp != 0 {
							if err := s.Dao.DeleteTaskExecution(&e); err != nil {
								log.Error(ctx, "deleteTaskExecutionsRoutine > error on DeleteTaskExecution: %v", err)
							}
							taskToDelete = true
						}
					default:
						if i >= s.Cfg.ExecutionHistory && e.ProcessingTimestamp != 0 {
							if err := s.Dao.DeleteTaskExecution(&e); err != nil {
								log.Error(ctx, "deleteTaskExecutionsRoutine > error on DeleteTaskExecution: %v", err)
							}
						}
					}

				}

				if taskToDelete {
					if err := s.deleteTask(ctx, &t); err != nil {
						log.Error(ctx, "deleteTaskExecutionsRoutine > Unable to deleteTask (%s): %v", t.UUID, err)
					}
				}
			}
		}
	}
}

// Get from queue task execution
func (s *Service) dequeueTaskExecutions(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		size, err := s.Dao.QueueLen()
		if err != nil {
			log.Error(ctx, "dequeueTaskExecutions > Unable to get queueLen: %v", err)
			continue
		}
		log.Debug("dequeueTaskExecutions> current queue size: %d", size)

		if s.Maintenance {
			log.Info(ctx, "Maintenance enable, wait 1 minute. Queue %d", size)
			time.Sleep(1 * time.Minute)
			continue
		}

		// Dequeuing context
		var taskKey string
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.Cache.DequeueWithContext(ctx, schedulerQueueKey, 250*time.Millisecond, &taskKey); err != nil {
			continue
		}
		s.Dao.dequeuedIncr()
		if taskKey == "" {
			continue
		}
		log.Info(ctx, "dequeueTaskExecutions> work on taskKey: %s", taskKey)

		// Load the task execution
		var t = sdk.TaskExecution{}
		find, err := s.Cache.Get(taskKey, &t)
		if err != nil {
			log.Error(ctx, "cannot get from cache %s: %v", taskKey, err)
		}
		if !find {
			continue
		}
		t.ProcessingTimestamp = time.Now().UnixNano()
		t.LastError = ""
		t.Status = TaskExecutionDoing
		s.Dao.SaveTaskExecution(&t)

		var restartTask bool
		var saveTaskExecution bool

		task := s.Dao.FindTask(ctx, t.UUID)
		if task == nil {
			log.Error(ctx, "dequeueTaskExecutions failed: Task %s not found - deleting this task execution", t.UUID)
			t.LastError = "Internal Error: Task not found"
			t.NbErrors++
			if err := s.Dao.DeleteTaskExecution(&t); err != nil {
				log.Error(ctx, "dequeueTaskExecutions > error on DeleteTaskExecution: %v", err)
			}
			continue

		} else if t.NbErrors >= s.Cfg.RetryError {
			log.Info(ctx, "dequeueTaskExecutions> Deleting task execution %s cause: to many errors:%d lastError:%s", t.UUID, t.NbErrors, t.LastError)
			if err := s.Dao.DeleteTaskExecution(&t); err != nil {
				log.Error(ctx, "dequeueTaskExecutions > error on DeleteTaskExecution: %v", err)
			}
			continue

		} else if task.Stopped {
			t.LastError = "Executions skipped: Task has been stopped"
			t.NbErrors++
			saveTaskExecution = true
		} else {
			saveTaskExecution = true
			log.Debug("dequeueTaskExecutions> call doTask on taskKey: %s", taskKey)
			var err error
			restartTask, err = s.doTask(ctx, task, &t)
			if err != nil {
				if strings.Contains(err.Error(), "Unsupported task type") {
					// delete this task execution, as it will never work
					log.Info(ctx, "dequeueTaskExecutions> Deleting task execution %s as err:%v", t.UUID, err)
					if err := s.Dao.DeleteTaskExecution(&t); err != nil {
						log.Error(ctx, "dequeueTaskExecutions > error on DeleteTaskExecution: %v", err)
					}
					continue
				} else {
					log.Warning(ctx, "dequeueTaskExecutions> %s failed err[%d]: %v", t.UUID, t.NbErrors, err)
					t.LastError = err.Error()
					t.NbErrors++
					saveTaskExecution = true
				}
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
			_, err := s.startTask(ctx, task)
			if err != nil {
				log.Error(ctx, "dequeueTaskExecutions> unable to restart the task %+v after execution: %v", task, err)
			}
		}
	}
}
