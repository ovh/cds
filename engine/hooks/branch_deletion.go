package hooks

import (
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) doBranchDeletionTaskExecution(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing branch deletion task %s", t.UUID)

	projectKey := t.Config["project"].Value
	workflowName := t.Config["workflow"].Value
	branch := t.Config["branch"].Value
	err := s.Client.WorkflowRunsDeleteByBranch(projectKey, workflowName, branch)

	return nil, sdk.WrapError(err, "cannot mark to delete workflow runs")
}

func (s *Service) stopBranchDeletionTask(branch string) error {
	keys, err := s.Dao.FindAllKeysMatchingPattern(branch + "*")
	if err != nil {
		return sdk.WrapError(err, "cannot find keys matching pattern %s", branch+"*")
	}
	for _, key := range keys {
		t := s.Dao.FindTask(key)
		if t == nil || t.Type != TypeBranchDeletion {
			continue
		}
		if err := s.stopTask(t); err != nil {
			log.Error("cannot stop task %s : %v", t.UUID, err)
		}
	}

	return nil
}
