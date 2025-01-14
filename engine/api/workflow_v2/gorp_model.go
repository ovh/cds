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
	var _ = []interface{}{r.ID, r.ProjectKey, r.VCSServerID, r.VCSServer, r.RepositoryID, r.Repository, r.WorkflowData, r.DeprecatedUserID}
	return gorpmapper.CanonicalForms{
		//"{{.ID}}{{.ProjectKey}}{{.VCSServerID}}{{.VCSServer}}{{.RepositoryID}}{{.Repository}}{{md5sum .WorkflowData}}{{md5sum .Initiator}}",
		"{{.ID}}{{.ProjectKey}}{{.VCSServerID}}{{.VCSServer}}{{.RepositoryID}}{{.Repository}}{{md5sum .WorkflowData}}{{.DeprecatedUserID}}",
		// TODO add context
	}
}

type dbWorkflowRunJob struct {
	sdk.V2WorkflowRunJob
	gorpmapper.SignedEntity
}

func (r dbWorkflowRunJob) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{r.ID, r.WorkflowRunID, r.ProjectKey, r.JobID, r.Job, r.DeprecatedUserID, r.Region, r.HatcheryName, r.Matrix}
	return gorpmapper.CanonicalForms{
		"{{.ID}}{{.WorkflowRunID}}{{.ProjectKey}}{{.JobID}}{{md5sum .Job}}{{.DeprecatedUserID}}{{.Region}}{{.HatcheryName}}{{md5sum .Matrix}}",
		"{{.ID}}{{.WorkflowRunID}}{{.ProjectKey}}{{.JobID}}{{hash .Job}}{{.DeprecatedUserID}}{{.Region}}{{.HatcheryName}}{{hash .Matrix}}",
	}
}

type dbWorkflowHook struct {
	sdk.V2WorkflowHook
	gorpmapper.SignedEntity
}

func (r dbWorkflowHook) Canonical() gorpmapper.CanonicalForms {
	var _ = []interface{}{r.ID, r.ProjectKey, r.VCSName, r.RepositoryName, r.EntityID, r.WorkflowName, r.Ref, r.Commit}
	return gorpmapper.CanonicalForms{
		"{{.ID}}{{.ProjectKey}}{{.VCSName}}{{.RepositoryName}}{{.EntityID}}{{.WorkflowName}}{{.Ref}}{{.Commit}}",
		// TODO add data
	}
}

type dbWorkflowRunInfo struct {
	sdk.V2WorkflowRunInfo
}

type dbWorkflowRunJobInfo struct {
	sdk.V2WorkflowRunJobInfo
}

type dbV2WorkflowRunResult struct {
	sdk.V2WorkflowRunResult
}

type dbV2WorkflowVersion struct {
	sdk.V2WorkflowVersion
}

func init() {
	gorpmapping.Register(gorpmapping.New(dbWorkflowRun{}, "v2_workflow_run", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbWorkflowRunJob{}, "v2_workflow_run_job", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbWorkflowRunInfo{}, "v2_workflow_run_info", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbWorkflowRunJobInfo{}, "v2_workflow_run_job_info", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbWorkflowHook{}, "v2_workflow_hook", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbV2WorkflowRunResult{}, "v2_workflow_run_result", false, "id"))
	gorpmapping.Register(gorpmapping.New(dbV2WorkflowVersion{}, "v2_workflow_version", false, "id"))
}
