package hooks

import (
	"context"
	"strings"
	"time"

	"github.com/ovh/cds/sdk"
	"github.com/rockbears/log"
)

const schedulerResyncLockKey = "hooks:v2:scheduler:resync:lock"

// resyncSchedulers reconciles the scheduler definitions in Redis with the source of truth in the API database.
// It adds missing schedulers, removes orphans, and updates schedulers whose configuration has changed.
func (s *Service) resyncSchedulers(ctx context.Context) error {
	// Acquire a distributed lock to prevent concurrent resync from multiple hook service instances
	lockAcquired, err := s.Dao.store.Lock(schedulerResyncLockKey, 5*time.Minute, 0, 1)
	if err != nil {
		return sdk.WrapError(err, "unable to acquire resync lock")
	}
	if !lockAcquired {
		log.Info(ctx, "resyncSchedulers> another instance is already resyncing, skipping")
		return nil
	}
	defer s.Dao.store.Unlock(schedulerResyncLockKey)

	t0 := time.Now()
	defer func() {
		log.Info(ctx, "resyncSchedulers> done (%.3fs)", time.Since(t0).Seconds())
	}()

	log.Info(ctx, "resyncSchedulers> starting scheduler resynchronization")

	// Step 1: Get the source of truth from the API database
	dbHooksByID, err := s.resyncLoadDBSchedulers(ctx)
	if err != nil {
		return err
	}

	// Step 2: Get current state from Redis (only parse keys, don't load full data)
	redisKeysByHookID, err := s.resyncLoadRedisSchedulerKeys(ctx)
	if err != nil {
		return err
	}

	// Step 3: Add missing schedulers and update those whose configuration has changed
	nbAdded, nbUpdated := s.resyncAddAndUpdateSchedulers(ctx, dbHooksByID, redisKeysByHookID)

	// Step 4: Remove orphan schedulers (in Redis but not in DB)
	nbRemoved := s.resyncRemoveOrphanSchedulers(ctx, dbHooksByID, redisKeysByHookID)

	// Step 5: Clean orphan executions (executions without a matching definition in DB)
	s.resyncCleanOrphanExecutions(ctx, dbHooksByID)

	// Step 6: Ensure every DB scheduler has a pending execution
	s.resyncEnsurePendingExecutions(ctx, dbHooksByID)

	log.Info(ctx, "resyncSchedulers> summary: added=%d removed=%d updated=%d (total in DB: %d, was in Redis: %d)",
		nbAdded, nbRemoved, nbUpdated, len(dbHooksByID), len(redisKeysByHookID))

	return nil
}

// resyncLoadDBSchedulers loads all scheduler hooks from the API database and returns them indexed by ID.
func (s *Service) resyncLoadDBSchedulers(ctx context.Context) (map[string]sdk.V2WorkflowHook, error) {
	dbHooks, err := s.Client.HookListAllSchedulerHooks(ctx)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list scheduler hooks from API")
	}
	dbHooksByID := make(map[string]sdk.V2WorkflowHook, len(dbHooks))
	for _, h := range dbHooks {
		dbHooksByID[h.ID] = h
	}
	return dbHooksByID, nil
}

// resyncLoadRedisSchedulerKeys lists all scheduler definition keys from Redis and returns a map hookID -> key.
// Keys are only parsed, not deserialized, to avoid loading unnecessary data.
func (s *Service) resyncLoadRedisSchedulerKeys(ctx context.Context) (map[string]string, error) {
	redisKeys, err := s.Dao.AllSchedulerKeys(ctx)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to list scheduler keys from Redis")
	}
	// Key format: hooks:v2:definition:schedulers:<vcs>:<repo>:<workflow>:<hookID>
	redisKeysByHookID := make(map[string]string, len(redisKeys))
	for _, k := range redisKeys {
		parts := strings.Split(k, ":")
		if len(parts) != 8 {
			log.Warn(ctx, "resyncSchedulers> invalid scheduler key: %s", k)
			continue
		}
		redisKeysByHookID[parts[7]] = k
	}
	return redisKeysByHookID, nil
}

// resyncAddAndUpdateSchedulers adds schedulers present in DB but missing from Redis,
// and updates those whose cron or timezone configuration has changed.
// Returns the number of added and updated schedulers.
func (s *Service) resyncAddAndUpdateSchedulers(ctx context.Context, dbHooksByID map[string]sdk.V2WorkflowHook, redisKeysByHookID map[string]string) (nbAdded, nbUpdated int) {
	for id, dbHook := range dbHooksByID {
		redisKey, exists := redisKeysByHookID[id]
		if !exists {
			// Scheduler is in DB but not in Redis: add it
			log.Info(ctx, "resyncSchedulers> adding missing scheduler %s (%s/%s/%s cron=%s tz=%s)",
				id, dbHook.VCSName, dbHook.RepositoryName, dbHook.WorkflowName, dbHook.Data.Cron, dbHook.Data.CronTimeZone)

			if err := s.Dao.CreateSchedulerDefinition(ctx, dbHook); err != nil {
				log.Error(ctx, "resyncSchedulers> unable to create scheduler definition %s: %v", id, err)
				continue
			}
			if err := s.createSchedulerNextExecution(ctx, dbHook); err != nil {
				log.Error(ctx, "resyncSchedulers> unable to create next execution for scheduler %s: %v", id, err)
				continue
			}
			nbAdded++
		} else {
			// Scheduler exists in Redis: load it to compare configuration
			var redisHook sdk.V2WorkflowHook
			found, err := s.Dao.store.Get(redisKey, &redisHook)
			if err != nil {
				log.Error(ctx, "resyncSchedulers> unable to read Redis key %s: %v", redisKey, err)
				continue
			}
			if !found {
				// Key disappeared between listing and reading: recreate definition + execution
				log.Info(ctx, "resyncSchedulers> Redis key vanished for scheduler %s, recreating", id)
				if err := s.Dao.CreateSchedulerDefinition(ctx, dbHook); err != nil {
					log.Error(ctx, "resyncSchedulers> unable to create scheduler definition %s: %v", id, err)
					continue
				}
				if err := s.createSchedulerNextExecution(ctx, dbHook); err != nil {
					log.Error(ctx, "resyncSchedulers> unable to create next execution for scheduler %s: %v", id, err)
					continue
				}
				nbAdded++
				continue
			}

			// Reload the latest version from the API to compare with Redis
			freshHook, err := s.Client.HookGetWorkflowHook(ctx, id)
			if err != nil {
				log.Error(ctx, "resyncSchedulers> unable to reload hook %s from API: %v", id, err)
				continue
			}

			if freshHook.Data.Cron != redisHook.Data.Cron || freshHook.Data.CronTimeZone != redisHook.Data.CronTimeZone {
				log.Info(ctx, "resyncSchedulers> updating scheduler %s (%s/%s/%s cron: %s->%s tz: %s->%s)",
					id, freshHook.VCSName, freshHook.RepositoryName, freshHook.WorkflowName,
					redisHook.Data.Cron, freshHook.Data.Cron,
					redisHook.Data.CronTimeZone, freshHook.Data.CronTimeZone)

				if err := s.Dao.CreateSchedulerDefinition(ctx, *freshHook); err != nil {
					log.Error(ctx, "resyncSchedulers> unable to update scheduler definition %s: %v", id, err)
					continue
				}
				if err := s.Dao.RemoveSchedulerExecution(ctx, id); err != nil {
					log.Error(ctx, "resyncSchedulers> unable to remove old execution for scheduler %s: %v", id, err)
					continue
				}
				if err := s.createSchedulerNextExecution(ctx, *freshHook); err != nil {
					log.Error(ctx, "resyncSchedulers> unable to create next execution for scheduler %s: %v", id, err)
					continue
				}
				nbUpdated++
			}
		}
	}
	return nbAdded, nbUpdated
}

// resyncRemoveOrphanSchedulers removes schedulers present in Redis but no longer in the API database.
// Each removal is double-checked with the API to avoid deleting recently created hooks.
// Returns the number of removed schedulers.
func (s *Service) resyncRemoveOrphanSchedulers(ctx context.Context, dbHooksByID map[string]sdk.V2WorkflowHook, redisKeysByHookID map[string]string) (nbRemoved int) {
	for hookID, redisKey := range redisKeysByHookID {
		if _, exists := dbHooksByID[hookID]; exists {
			continue
		}
		// Double-check with the API that the hook truly doesn't exist anymore.
		// A new hook may have been created after our initial snapshot.
		if _, err := s.Client.HookGetWorkflowHook(ctx, hookID); err == nil {
			log.Info(ctx, "resyncSchedulers> scheduler %s not in initial snapshot but exists in API, skipping removal", hookID)
			continue
		}
		parts := strings.Split(redisKey, ":")
		if len(parts) != 8 {
			log.Warn(ctx, "resyncSchedulers> invalid scheduler key: %s", redisKey)
			continue
		}
		vcs, repo, workflow := parts[4], parts[5], parts[6]
		log.Info(ctx, "resyncSchedulers> removing orphan scheduler %s (%s/%s/%s)", hookID, vcs, repo, workflow)

		if err := s.Dao.RemoveScheduler(ctx, vcs, repo, workflow, hookID); err != nil {
			log.Error(ctx, "resyncSchedulers> unable to remove orphan scheduler %s: %v", hookID, err)
			continue
		}
		nbRemoved++
	}
	return nbRemoved
}

// resyncCleanOrphanExecutions removes scheduled executions whose hook no longer exists in the API database.
func (s *Service) resyncCleanOrphanExecutions(ctx context.Context, dbHooksByID map[string]sdk.V2WorkflowHook) {
	allExecutions, err := s.Dao.GetAllSchedulerExecutions(ctx)
	if err != nil {
		log.Error(ctx, "resyncSchedulers> unable to load scheduler executions: %v", err)
		return
	}
	for _, exec := range allExecutions {
		if _, exists := dbHooksByID[exec.SchedulerDef.ID]; exists {
			continue
		}
		// Double-check with the API before removing
		if _, err := s.Client.HookGetWorkflowHook(ctx, exec.SchedulerDef.ID); err == nil {
			continue
		}
		log.Info(ctx, "resyncSchedulers> removing orphan execution for hook %s", exec.SchedulerDef.ID)
		if err := s.Dao.RemoveSchedulerExecution(ctx, exec.SchedulerDef.ID); err != nil {
			log.Error(ctx, "resyncSchedulers> unable to remove orphan execution %s: %v", exec.SchedulerDef.ID, err)
		}
	}
}

// resyncEnsurePendingExecutions checks that every scheduler in the DB has a pending execution in Redis.
// If an execution is missing, it is recreated.
func (s *Service) resyncEnsurePendingExecutions(ctx context.Context, dbHooksByID map[string]sdk.V2WorkflowHook) {
	for id, dbHook := range dbHooksByID {
		exec, err := s.Dao.GetSchedulerExecution(ctx, id)
		if err != nil {
			log.Error(ctx, "resyncSchedulers> unable to check execution for scheduler %s: %v", id, err)
			continue
		}
		if exec == nil {
			log.Info(ctx, "resyncSchedulers> recreating missing execution for scheduler %s", id)
			if err := s.createSchedulerNextExecution(ctx, dbHook); err != nil {
				log.Error(ctx, "resyncSchedulers> unable to recreate execution for scheduler %s: %v", id, err)
			}
		}
	}
}
