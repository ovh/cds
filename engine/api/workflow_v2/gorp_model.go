package workflow_v2

import (
  "github.com/ovh/cds/engine/api/database/gorpmapping"
  "github.com/ovh/cds/engine/gorpmapper"
  "github.com/ovh/cds/sdk"
)

type dbWorkflowRun struct {
  sdk.V2WorkflowRun
  gorpmapper.SignedEntity
}

func (r dbWorkflowRun) Canonical() gorpmapper.CanonicalForms {
  var _ = []interface{}{r.ID, r.ProjectKey, r.VCSServerID, r.RepositoryID, r.WorkflowName, r.WorkflowData, r.UserID, r.Contexts}
  return gorpmapper.CanonicalForms{
    "{{.ID}}{{.ProjectKey}}{{.VCSServerID}}{{.RepositoryID}}{{.WorkflowData}}{{.UserID}}{{.Contexts}}",
  }
}

type dbWorkflowRunJob struct {
  sdk.V2WorkflowRunJob
  gorpmapper.SignedEntity
}

func (r dbWorkflowRunJob) Canonical() gorpmapper.CanonicalForms {
  var _ = []interface{}{r.ID, r.WorkflowRunID, r.JobID, r.Job, r.Outputs}
  return gorpmapper.CanonicalForms{
    "{{.ID}}{{.WorkflowRunID}}{{.JobID}}{{.Job}}{{.Outputs}}",
  }
}

type dbWorkflowRunInfo struct {
  sdk.V2WorkflowRunInfo
}

func init() {
  gorpmapping.Register(gorpmapping.New(dbWorkflowRun{}, "v2_workflow_run", false, "id"))
  gorpmapping.Register(gorpmapping.New(dbWorkflowRunJob{}, "v2_workflow_run_job", false, "id"))
  gorpmapping.Register(gorpmapping.New(dbWorkflowRunInfo{}, "v2_workflow_run_info", false, "id"))
}
