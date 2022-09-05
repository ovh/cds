package sdk

import (
	"github.com/rockbears/yaml"
	"time"
)

const (
	EntityTypeWorkerModelTemplate = "WorkerModelTemplate"
	EntityTypeWorkerModel         = "WorkerModel"
)

type Entity struct {
	ID                  string    `json:"id" db:"id"`
	ProjectKey          string    `json:"project_key" db:"project_key"`
	ProjectRepositoryID string    `json:"project_repository_id" db:"project_repository_id"`
	Type                string    `json:"type" db:"type"`
	Name                string    `json:"name" db:"name"`
	Branch              string    `json:"branch" db:"branch"`
	Commit              string    `json:"commit" db:"commit"`
	LastUpdate          time.Time `json:"last_update" db:"last_update"`
	Data                string    `json:"data" db:"data"`
}

type Lintable interface {
	Lint() error
	GetName() string
}

func ReadEntityFile[T Lintable](directory, fileName string, content []byte, out *[]T, t string, analysis ProjectRepositoryAnalysis) ([]Entity, error) {
	if err := yaml.UnmarshalMultipleDocuments(content, out); err != nil {
		return nil, WrapError(err, "unable to read %s%s", directory, fileName)
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
		})
	}
	return entities, nil
}
