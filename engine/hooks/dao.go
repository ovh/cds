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
	nbTasks := d.store.SetCard(rootKey)
	tasks := make([]*sdk.Task, nbTasks, nbTasks)
	for i := 0; i < nbTasks; i++ {
		tasks[i] = &sdk.Task{}
	}
	if err := d.store.SetScan(rootKey, sdk.InterfaceSlice(tasks)...); err != nil {
		return nil, sdk.WrapError(err, "hooks>FindAllLongRunningTasks> Unable to scan %s", rootKey)
	}

	alltasks := make([]sdk.Task, nbTasks)
	for i := 0; i < nbTasks; i++ {
		alltasks[i] = *tasks[i]
	}

	return alltasks, nil
}

func (d *dao) FindTask(uuid string) *sdk.Task {
	key := cache.Key(rootKey, uuid)
	t := &sdk.Task{}
	if d.store.Get(key, t) {
		return t
	}
	return nil
}

func (d *dao) SaveTask(r *sdk.Task) {
	d.store.SetAdd(rootKey, r.UUID, r)
}

func (d *dao) DeleteTask(r *sdk.Task) {
	d.store.SetRemove(rootKey, r.UUID, r)
	execs, _ := d.FindAllTaskExecutions(r)
	for _, e := range execs {
		d.DeleteTaskExecution(&e)
	}
}

func (d *dao) SaveTaskExecution(r *sdk.TaskExecution) {
	setKey := cache.Key(executionRootKey, r.Type, r.UUID)
	execKey := fmt.Sprintf("%d", r.Timestamp)
	d.store.SetAdd(setKey, execKey, r)
}

func (d *dao) DeleteTaskExecution(r *sdk.TaskExecution) {
	setKey := cache.Key(executionRootKey, r.Type, r.UUID)
	execKey := fmt.Sprintf("%d", r.Timestamp)
	d.store.SetRemove(setKey, execKey, r)
}

func (d *dao) EnqueueTaskExecution(r *sdk.TaskExecution) {
	k := cache.Key(executionRootKey, r.Type, r.UUID, fmt.Sprintf("%d", r.Timestamp))
	d.store.Enqueue(schedulerQueueKey, k)
}

func (d *dao) QueueLen() int {
	return d.store.QueueLen(schedulerQueueKey)
}

func (d *dao) FindAllTaskExecutions(t *sdk.Task) ([]sdk.TaskExecution, error) {
	nbExecutions := d.store.SetCard(cache.Key(executionRootKey, t.Type, t.UUID))
	execs := make([]*sdk.TaskExecution, nbExecutions, nbExecutions)
	for i := 0; i < nbExecutions; i++ {
		execs[i] = &sdk.TaskExecution{}
	}
	if err := d.store.SetScan(cache.Key(executionRootKey, t.Type, t.UUID), sdk.InterfaceSlice(execs)...); err != nil {
		return nil, sdk.WrapError(err, "hooks>FindAllTaskExecutions> Unable to scan %s", rootKey)
	}

	allexecs := make([]sdk.TaskExecution, nbExecutions)
	for i := 0; i < nbExecutions; i++ {
		allexecs[i] = *execs[i]
	}

	return allexecs, nil
}
