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
