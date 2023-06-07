package sdk

import (
	"regexp"
	"time"

	"github.com/rockbears/yaml"
)

const (
	EntityTypeWorkerModel = "WorkerModel"
	EntityTypeAction      = "Action"
	EntityTypeWorkflow    = "Workflow"
	EntityNamePattern     = "^[a-zA-Z0-9._-]{1,}$"
)

type EntityFullName struct {
	Name       string `json:"name" db:"name"`
	Branch     string `json:"branch" db:"branch"`
	VCSName    string `json:"vcs_name" db:"vcs_name"`
	RepoName   string `json:"repo_name" db:"repo_name"`
	ProjectKey string `json:"project_key" db:"project_key"`
}

type ShortEntity struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Branch string `json:"branch"`
}

type Entity struct {
	ID                  string    `json:"id" db:"id"`
	ProjectKey          string    `json:"project_key" db:"project_key"`
	ProjectRepositoryID string    `json:"project_repository_id" db:"project_repository_id"`
	Type                string    `json:"type" db:"type"`
	FilePath            string    `json:"file_path" db:"file_path"`
	Name                string    `json:"name" db:"name"`
	Branch              string    `json:"branch" db:"branch"`
	Commit              string    `json:"commit" db:"commit"`
	LastUpdate          time.Time `json:"last_update" db:"last_update"`
	Data                string    `json:"data" db:"data"`
}

func GetManageRoleByEntity(entityType string) (string, error) {
	switch entityType {
	case EntityTypeWorkerModel:
		return ProjectRoleManageWorkerModel, nil
	case EntityTypeAction:
		return ProjectRoleManageAction, nil
  case EntityTypeWorkflow:
    return ProjectRoleManageWorkflow, nil
	}
	return "", NewErrorFrom(ErrInvalidData, "unknown entity of type %s", entityType)
}

type Lintable interface {
	Lint() []error
	GetName() string
}

func ReadEntityFile[T Lintable](directory, fileName string, content []byte, out *[]T, t string, analysis ProjectRepositoryAnalysis) ([]Entity, []error) {
	namePattern, err := regexp.Compile(EntityNamePattern)
	if err != nil {
		return nil, []error{WrapError(err, "unable to compile regexp %s", namePattern)}
	}

	if err := yaml.UnmarshalMultipleDocuments(content, out); err != nil {
		return nil, []error{NewErrorFrom(ErrInvalidData, "unable to read %s%s: %v", directory, fileName, err)}
	}
	var entities []Entity
	for _, o := range *out {
		if err := o.Lint(); err != nil {
			return nil, err
		}
		entities = append(entities, Entity{
			Data:                string(content),
			Name:                o.GetName(),
			Branch:              analysis.Branch,
			Commit:              analysis.Commit,
			ProjectKey:          analysis.ProjectKey,
			ProjectRepositoryID: analysis.ProjectRepositoryID,
			Type:                t,
			FilePath:            directory + fileName,
		})
		if !namePattern.MatchString(o.GetName()) {
			return nil, []error{WrapError(ErrInvalidData, "name %s doesn't match %s", o.GetName(), EntityNamePattern)}
		}
	}
	return entities, nil
}
