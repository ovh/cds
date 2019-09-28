package hooks

import (
	"fmt"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

type dao struct {
	store cache.Store
}

func (d *dao) FindAllTasks() ([]sdk.Task, error) {
	nbTasks, err := d.store.SetCard(rootKey)
	if err != nil {
		return nil, sdk.WrapError(err, "unsable to setCard %v", rootKey)
	}
	tasks := make([]*sdk.Task, nbTasks, nbTasks)
	for i := 0; i < nbTasks; i++ {
		tasks[i] = &sdk.Task{}
	}
	if err := d.store.SetScan(rootKey, sdk.InterfaceSlice(tasks)...); err != nil {
		return nil, sdk.WrapError(err, "Unable to scan %s", rootKey)
	}

	alltasks := make([]sdk.Task, nbTasks)
	for i := 0; i < nbTasks; i++ {
		alltasks[i] = *tasks[i]
	}

	return alltasks, nil
}

func (d *dao) FindAllKeysMatchingPattern(pattern string) ([]string, error) {
	return d.store.ZScan(rootKey, pattern)
}

func (d *dao) FindTask(uuid string) *sdk.Task {
	key := cache.Key(rootKey, uuid)
	t := &sdk.Task{}
	if d.store.Get(key, t) {
		return t
	}
	return nil
}

func (d *dao) SaveTask(r *sdk.Task) error {
	return d.store.SetAdd(rootKey, r.UUID, r)
}

func (d *dao) DeleteTask(r *sdk.Task) error {
	if err := d.store.SetRemove(rootKey, r.UUID, r); err != nil {
		return err
	}
	execs, _ := d.FindAllTaskExecutions(r)
	for _, e := range execs {
		if err := d.DeleteTaskExecution(&e); err != nil {
			return err
		}
	}
	return nil
}

func (d *dao) SaveTaskExecution(r *sdk.TaskExecution) error {
	setKey := cache.Key(executionRootKey, r.Type, r.UUID)
	execKey := fmt.Sprintf("%d", r.Timestamp)
	return d.store.SetAdd(setKey, execKey, r)
}

func (d *dao) DeleteTaskExecution(r *sdk.TaskExecution) error {
	setKey := cache.Key(executionRootKey, r.Type, r.UUID)
	execKey := fmt.Sprintf("%d", r.Timestamp)
	return d.store.SetRemove(setKey, execKey, r)
}

func (d *dao) EnqueueTaskExecution(r *sdk.TaskExecution) error {
	k := cache.Key(executionRootKey, r.Type, r.UUID, fmt.Sprintf("%d", r.Timestamp))
	// before enqueue, be sure that it's not in queue
	if err := d.store.RemoveFromQueue(schedulerQueueKey, k); err != nil {
		return err
	}
	return d.store.Enqueue(schedulerQueueKey, k)
}

func (d *dao) QueueLen() (int, error) {
	return d.store.QueueLen(schedulerQueueKey)
}

func (d *dao) FindAllTaskExecutions(t *sdk.Task) ([]sdk.TaskExecution, error) {
	nbExecutions, err := d.store.SetCard(cache.Key(executionRootKey, t.Type, t.UUID))
	if err != nil {
		return nil, sdk.WrapError(err, "unable to setCard %s", cache.Key(executionRootKey, t.Type, t.UUID))
	}
	execs := make([]*sdk.TaskExecution, nbExecutions, nbExecutions)
	for i := 0; i < nbExecutions; i++ {
		execs[i] = &sdk.TaskExecution{}
	}
	if err := d.store.SetScan(cache.Key(executionRootKey, t.Type, t.UUID), sdk.InterfaceSlice(execs)...); err != nil {
		return nil, sdk.WrapError(err, "Unable to scan %s", rootKey)
	}

	allexecs := make([]sdk.TaskExecution, nbExecutions)
	for i := 0; i < nbExecutions; i++ {
		allexecs[i] = *execs[i]
	}

	return allexecs, nil
}

func (d *dao) FindAllTaskExecutionsForTasks(ts ...sdk.Task) ([]sdk.TaskExecution, error) {
	var tes []sdk.TaskExecution
	for _, t := range ts {
		res, err := d.FindAllTaskExecutions(&t)
		if err != nil {
			return nil, err
		}
		tes = append(tes, res...)
	}

	return tes, nil
}
