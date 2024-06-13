package hooks

import (
	"context"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
)

// Scheduler definition

func GetSchedulerDefinitionKey(vcs, repo, workflow, whID string) string {
	return cache.Key(scheduleDefinitionRootKey, vcs, repo, workflow, whID)
}

// SchedulerKeysByWorkflow returns all the scheduler definition keys for the given workflow
func (d *dao) SchedulerKeysByWorkflow(ctx context.Context, vcs, repo, workflow string) ([]string, error) {
	return d.store.Keys(cache.Key(scheduleDefinitionRootKey, vcs, repo, workflow))
}

func (d *dao) GetSchedulerDefinition(ctx context.Context, vcs, repo, workflow, whID string) (*sdk.V2WorkflowHook, error) {
	var h sdk.V2WorkflowHook
	found, err := d.store.Get(GetSchedulerDefinitionKey(vcs, repo, workflow, whID), &h)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &h, nil
}

// RemoveScheduler removes the next execution + the definition of the given scheduler
func (d *dao) RemoveScheduler(ctx context.Context, vcs, repo, workflow, whID string) error {
	// Remove the next execution
	if err := d.RemoveSchedulerExecution(ctx, whID); err != nil {
		return err
	}
	//Remove the definition
	return d.store.Delete(GetSchedulerDefinitionKey(vcs, repo, workflow, whID))
}

func (d *dao) RemoveSchedulerExecution(ctx context.Context, whID string) error {
	return d.store.Delete(cache.Key(schedulerNextExecutionRootKey, whID))
}

func (d *dao) CreateSchedulerDefinition(ctx context.Context, h sdk.V2WorkflowHook) error {
	if err := d.store.SetWithTTL(GetSchedulerDefinitionKey(h.VCSName, h.RepositoryName, h.WorkflowName, h.ID), h, 0); err != nil {
		return err
	}
	return nil
}

func (d *dao) CreateSchedulerNextExecution(ctx context.Context, exec SchedulerExecution) error {
	if err := d.RemoveSchedulerExecution(ctx, exec.SchedulerDef.ID); err != nil {
		return err
	}
	if err := d.store.SetAdd(schedulerNextExecutionRootKey, exec.SchedulerDef.ID, exec); err != nil {
		return err
	}
	return nil
}

func (d *dao) GetAllSchedulerExecutions(ctx context.Context) ([]SchedulerExecution, error) {
	nbExec, err := d.store.SetCard(schedulerNextExecutionRootKey)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to setCard %v", schedulerNextExecutionRootKey)
	}
	schedulerExecs := make([]*SchedulerExecution, nbExec, nbExec)
	for i := 0; i < nbExec; i++ {
		schedulerExecs[i] = &SchedulerExecution{}
	}
	if err := d.store.SetScan(ctx, schedulerNextExecutionRootKey, sdk.InterfaceSlice(schedulerExecs)...); err != nil {
		return nil, sdk.WrapError(err, "Unable to scan %s", schedulerNextExecutionRootKey)
	}

	allExec := make([]SchedulerExecution, nbExec)
	for i := 0; i < nbExec; i++ {
		allExec[i] = *schedulerExecs[i]
	}
	return allExec, nil
}

func (d *dao) GetSchedulerExecution(ctx context.Context, hookID string) (*SchedulerExecution, error) {
	var e SchedulerExecution
	found, err := d.store.Get(cache.Key(schedulerNextExecutionRootKey, hookID), &e)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, nil
	}
	return &e, nil
}
