package template

import (
	"database/sql"
	"fmt"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/log"
	"github.com/ovh/cds/sdk"
)

// UglyID OH I KNOW IT'S UGLY OK
const UglyID = 10000

func applyBuildTemplate(tx *sql.Tx, p *sdk.Project, app *sdk.Application) (*sdk.Pipeline, error) {
	switch app.BuildTemplate.ID {
	case UglyID:
		return nil, nil
	}
	return nil, sdk.ErrUnknownTemplate
}

func createBuildPipeline(tx *sql.Tx, name string, app *sdk.Application, p *sdk.Project, groups []sdk.GroupPermission) (*sdk.Pipeline, bool, error) {
	pip := &sdk.Pipeline{}
	projectID := p.ID
	var exists bool

	dbpip, err := pipeline.LoadPipeline(tx, p.Key, name, false)
	if err == nil {
		pip = dbpip
		exists = true
	} else {
		pip.Name = name
		pip.Type = sdk.BuildPipeline
		pip.ProjectID = projectID
		err := pipeline.InsertPipeline(tx, pip)
		if err != nil {
			return nil, exists, fmt.Errorf("createBuildPipeline> InsertPipeline> %s", err)
		}

		// Add groups
		err = group.InsertGroupsInPipeline(tx, groups, pip.ID)
		if err != nil {
			return nil, exists, fmt.Errorf("createBuildPipeline> InsertGroupsInPipeline> %s\n", err)
		}
	}

	// Attach pipeline to app
	err = application.AttachPipeline(tx, app.ID, pip.ID)
	if err != nil {
		return nil, exists, fmt.Errorf("createBuildPipeline> Attach pipeline> %s", err)
	}
	app.Pipelines = append(app.Pipelines, sdk.ApplicationPipeline{Pipeline: *pip})

	return pip, exists, nil
}

func createPackagingPipeline(tx *sql.Tx, name string, app *sdk.Application, p *sdk.Project, groups []sdk.GroupPermission) (*sdk.Pipeline, bool, error) {
	pip := &sdk.Pipeline{}
	projectID := p.ID
	var exists bool

	dbpip, err := pipeline.LoadPipeline(tx, p.Key, name, false)
	if err == nil {
		pip = dbpip
		exists = true
	} else {
		pip.Name = name
		pip.Type = sdk.BuildPipeline
		pip.ProjectID = projectID
		err := pipeline.InsertPipeline(tx, pip)
		if err != nil {
			return nil, false, fmt.Errorf("createPackagingPipeline> InsertPipeline> %s", err)
		}

		// Add groups
		err = group.InsertGroupsInPipeline(tx, groups, pip.ID)
		if err != nil {
			return nil, false, fmt.Errorf("createPackagingPipeline> InsertGroupsInPipeline> %s\n", err)
		}
	}

	// Attach pipeline to app
	err = application.AttachPipeline(tx, app.ID, pip.ID)
	if err != nil {
		return nil, false, fmt.Errorf("createPackagingPipeline> Attach pipeline> %s", err)
	}
	app.Pipelines = append(app.Pipelines, sdk.ApplicationPipeline{Pipeline: *pip})

	return pip, exists, nil
}

func createDeploymentPipeline(tx *sql.Tx, name string, app *sdk.Application, p *sdk.Project, groups []sdk.GroupPermission) (*sdk.Pipeline, bool, error) {
	pip := &sdk.Pipeline{}
	projectID := p.ID
	var exists bool

	dbpip, err := pipeline.LoadPipeline(tx, p.Key, name, false)
	if err == nil {
		pip = dbpip
		exists = true
	} else {
		pip.Name = name
		pip.Type = sdk.DeploymentPipeline
		pip.ProjectID = projectID
		err := pipeline.InsertPipeline(tx, pip)
		if err != nil {
			return nil, false, fmt.Errorf("createDeploymentPipeline> InsertPipeline> %s", err)
		}

		// Add groups
		err = group.InsertGroupsInPipeline(tx, groups, pip.ID)
		if err != nil {
			return nil, false, fmt.Errorf("createDeploymentPipeline> InsertGroupsInPipeline> %s\n", err)
		}
	}

	// Attach pipeline to app
	err = application.AttachPipeline(tx, app.ID, pip.ID)
	if err != nil {
		return nil, false, fmt.Errorf("createDeploymentPipeline> Attach pipeline> %s", err)
	}
	app.Pipelines = append(app.Pipelines, sdk.ApplicationPipeline{Pipeline: *pip})

	return pip, exists, nil
}

func createStage(tx *sql.Tx, name string, pip *sdk.Pipeline, bo int) (*sdk.Stage, error) {
	s := &sdk.Stage{
		Name:       name,
		PipelineID: pip.ID,
		BuildOrder: bo,
		Enabled:    true,
	}
	err := pipeline.InsertStage(tx, s)
	if err != nil {
		log.Warning("createStage> InsertStage> %s", err)
		return nil, err
	}

	pip.Stages = append(pip.Stages, *s)
	return s, nil
}

func newJoinedAction(name, desc string) *sdk.Action {
	a := &sdk.Action{
		Name:        name,
		Type:        sdk.JoinedAction,
		Description: desc,
		Enabled:     true,
	}

	return a
}

func addScriptAction(tx *sql.Tx, parent *sdk.Action, value string) error {
	s, err := action.LoadPublicAction(tx, "Script")
	if err != nil {
		return fmt.Errorf("addScriptAction> LoadPublicAction script> %s", err)
	}

	for i := range s.Parameters {
		if s.Parameters[i].Name == "script" {
			s.Parameters[i].Value = value
		}
	}
	parent.Actions = append(parent.Actions, *s)

	return nil
}

func addJoinedAction(tx *sql.Tx, a *sdk.Action, p *sdk.Project, app *sdk.Application, pip *sdk.Pipeline, s *sdk.Stage) error {

	// Insert compilation action
	err := action.InsertAction(tx, a, false)
	if err != nil {
		return fmt.Errorf("addJoinedAction> Insert action> %s", err)
	}

	// Add compilation action
	_, err = pipeline.InsertPipelineAction(tx, p.Key, pip.Name, a.ID, "[]", s.ID)
	if err != nil {
		return fmt.Errorf("addJoinedAction> Insert pipeline action> %s", err)
	}
	s.Actions = append(s.Actions, *a)
	return nil
}

func addArtifactUploadAction(tx *sql.Tx, parent *sdk.Action, path, tag string) error {
	a, err := action.LoadPublicAction(tx, "Artifact Upload")
	if err != nil {
		return fmt.Errorf("addArtifactUploadAction> Loading Artifact Upload: %s", err)
	}
	for i := range a.Parameters {
		if a.Parameters[i].Name == "path" {
			a.Parameters[i].Value = path
		}
		if a.Parameters[i].Name == "tag" {
			a.Parameters[i].Value = tag
		}
	}

	parent.Actions = append(parent.Actions, *a)
	return nil
}

func addArtifactDownloadAction(tx *sql.Tx, parent *sdk.Action, application, pipeline, path, tag string) error {
	a, err := action.LoadPublicAction(tx, "Artifact Download")
	if err != nil {
		return fmt.Errorf("addArtifactDownloadAction> Loading Artifact Download: %s", err)
	}
	for i := range a.Parameters {
		if a.Parameters[i].Name == "pipeline" {
			a.Parameters[i].Value = pipeline
		}
		if a.Parameters[i].Name == "path" {
			a.Parameters[i].Value = path
		}
		if a.Parameters[i].Name == "tag" {
			a.Parameters[i].Value = tag
		}
		if a.Parameters[i].Name == "application" {
			a.Parameters[i].Value = application
		}
	}

	parent.Actions = append(parent.Actions, *a)
	return nil
}
