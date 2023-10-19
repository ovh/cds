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
		"{{.ID}}{{.ProjectKey}}{{.VCSServerID}}{{.RepositoryID}}{{hash .WorkflowData}}{{.UserID}}{{hash .Contexts}}",
	}
}

type dbWorkflowRunJob struct {
	sdk.V2WorkflowRunJob
	gorpmapper.SignedEntity
}

func (r dbWorkflowRunJob) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{r.ID, r.WorkflowRunID, r.ProjectKey, r.JobID, r.Job, r.Outputs, r.UserID, r.Region, r.HatcheryName, r.Matrix}
	return gorpmapper.CanonicalForms{
		"{{.ID}}{{.WorkflowRunID}}{{.ProjectKey}}{{.JobID}}{{hash .Job}}{{hash .Outputs}}{{.UserID}}{{.Region}}{{.HatcheryName}}{{hash .Matrix}}",
	}
}

type dbWorkflowHook struct {
	sdk.V2WorkflowHook
	gorpmapper.SignedEntity
}

func (r dbWorkflowHook) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{r.ID, r.Data, r.ProjectKey, r.VCSName, r.RepositoryName, r.EntityID, r.WorkflowName, r.Branch, r.Commit}
	return gorpmapper.CanonicalForms{
		"{{.ID}}{{.Data}}{{.ProjectKey}}{{.VCSName}}{{.RepositoryName}}{{.EntityID}}{{.WorkflowName}}{{.Branch}}{{.Commit}}",
	}
}

type dbWorkflowRunInfo struct {
	sdk.V2WorkflowRunInfo
}

type dbWorkflowRunJobInfo struct {
	sdk.V2WorkflowRunJobInfo
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbWorkflowRun{}, "v2_workflow_run", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbWorkflowRunJob{}, "v2_workflow_run_job", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbWorkflowRunInfo{}, "v2_workflow_run_info", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbWorkflowRunJobInfo{}, "v2_workflow_run_job_info", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbWorkflowHook{}, "v2_workflow_hook", false, "id"))
}
