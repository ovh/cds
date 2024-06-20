package sdk

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

const (
	EntityTypeWorkerModel      = "WorkerModel"
	EntityTypeAction           = "Action"
	EntityTypeWorkflow         = "Workflow"
	EntityTypeWorkflowTemplate = "WorkflowTemplate"
	EntityTypeJob              = "Job"
	EntityNamePattern          = "^[a-zA-Z0-9._-]{1,}$"
)

var EntityTypes = []string{EntityTypeWorkerModel, EntityTypeAction, EntityTypeWorkflow, EntityTypeWorkflowTemplate}

type EntityFullName struct {
	Name       string `json:"name" db:"name"`
	Ref        string `json:"ref" db:"ref"`
	VCSName    string `json:"vcs_name" db:"vcs_name"`
	RepoName   string `json:"repo_name" db:"repo_name"`
	ProjectKey string `json:"project_key" db:"project_key"`
}

type ShortEntity struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

type Entity struct {
	ID                  string    `json:"id" db:"id"`
	ProjectKey          string    `json:"project_key" db:"project_key"`
	ProjectRepositoryID string    `json:"project_repository_id" db:"project_repository_id"`
	Type                string    `json:"type" db:"type"`
	FilePath            string    `json:"file_path" db:"file_path"`
	Name                string    `json:"name" db:"name"`
	Commit              string    `json:"commit" db:"commit"`
	Ref                 string    `json:"ref" db:"ref"`
	LastUpdate          time.Time `json:"last_update" db:"last_update"`
	Data                string    `json:"data" db:"data"`
}

type EntityWithObject struct {
	Entity
	Workflow V2Workflow
	Action   V2Action
	Model    V2WorkerModel
	Template V2WorkflowTemplate
}

func (e *EntityWithObject) Interpolate(ctx context.Context) error {
	var refName string
	switch {
	case strings.HasPrefix(e.Entity.Ref, GitRefTagPrefix):
		refName = strings.TrimPrefix(e.Entity.Ref, GitRefTagPrefix)
	default:
		refName = strings.TrimPrefix(e.Entity.Ref, GitRefBranchPrefix)
	}
	ap := NewActionParser(map[string]interface{}{"git": map[string]interface{}{"ref_name": refName}}, nil)
	var err error
	switch e.Model.Type {
	case WorkerModelTypeDocker:
		var dockerSpec V2WorkerModelDockerSpec
		if err := json.Unmarshal(e.Model.Spec, &dockerSpec); err != nil {
			return WrapError(err, "unable to unmarshal docker spec")
		}
		dockerSpec.Image, err = ap.InterpolateToString(ctx, dockerSpec.Image)
		if err != nil {
			return WithStack(err)
		}
		e.Model.Spec, err = json.Marshal(dockerSpec)
		if err != nil {
			return WrapError(err, "unable to marshal docker spec")
		}
	case WorkerModelTypeVSphere:
		var vsphereSpec V2WorkerModelVSphereSpec
		if err := json.Unmarshal(e.Model.Spec, &vsphereSpec); err != nil {
			return WrapError(err, "unable to unmarshal vsphere spec")
		}
		vsphereSpec.Image, err = ap.InterpolateToString(ctx, vsphereSpec.Image)
		if err != nil {
			return WithStack(err)
		}
		e.Model.Spec, err = json.Marshal(vsphereSpec)
		if err != nil {
			return WrapError(err, "unable to marshal vsphere spec")
		}
	case WorkerModelTypeOpenstack:
		var openstackSpec V2WorkerModelOpenstackSpec
		if err := json.Unmarshal(e.Model.Spec, &openstackSpec); err != nil {
			return WrapError(err, "unable to unmarshal openstack spec")
		}
		openstackSpec.Image, err = ap.InterpolateToString(ctx, openstackSpec.Image)
		if err != nil {
			return WithStack(err)
		}
		e.Model.Spec, err = json.Marshal(openstackSpec)
		if err != nil {
			return WrapError(err, "unable to marshal openstack spec")
		}
	}
	return nil
}

func GetManageRoleByEntity(entityType string) (string, error) {
	switch entityType {
	case EntityTypeWorkerModel:
		return ProjectRoleManageWorkerModel, nil
	case EntityTypeAction:
		return ProjectRoleManageAction, nil
	case EntityTypeWorkflow:
		return ProjectRoleManageWorkflow, nil
	case EntityTypeWorkflowTemplate:
		return ProjectRoleManageWorkflowTemplate, nil
	}
	return "", NewErrorFrom(ErrInvalidData, "unknown entity of type %s", entityType)
}

type Lintable interface {
	Lint() []error
	GetName() string
}

type EntityCheckResponse struct {
	Messages []string `json:"messages"`
}
