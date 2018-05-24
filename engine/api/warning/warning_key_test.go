package warning

import (
	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_KeyUsedInApplication(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}
	u, _ := assets.InsertAdminUser(db)

	p := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, project.Insert(db, store, p, u))

	a := &sdk.Application{
		Name:      sdk.RandomString(10),
		ProjectID: p.ID,
		RepositoryStrategy: sdk.RepositoryStrategy{
			SSHKey: "proj-key",
		},
	}
	assert.NoError(t, application.Insert(db, store, p, a, u))

	k := sdk.ProjectKey{
		ProjectID: p.ID,
		Builtin:   false,
		Key: sdk.Key{
			Name: "proj-key",
		},
	}
	assert.NoError(t, project.InsertKey(db, &k))

	apps, pips := keyIsUsed(db, p.Key, "proj-key")
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, 0, len(pips))
}

func Test_KeyUsedInPipelineParam(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}
	u, _ := assets.InsertAdminUser(db)

	p := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, project.Insert(db, store, p, u))

	k := sdk.ProjectKey{
		ProjectID: p.ID,
		Builtin:   false,
		Key: sdk.Key{
			Name: "proj-key",
		},
	}
	assert.NoError(t, project.InsertKey(db, &k))

	pip := &sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: p.ID,
		Type:      "build",
	}
	assert.NoError(t, pipeline.InsertPipeline(db, store, p, pip, u))

	param := sdk.Parameter{
		Name:  sdk.RandomString(3),
		Type:  "ssh-key",
		Value: "proj-key",
	}
	assert.NoError(t, pipeline.InsertParameterInPipeline(db, pip.ID, &param))

	apps, pips := keyIsUsed(db, p.Key, "proj-key")
	assert.Equal(t, 0, len(apps))
	assert.Equal(t, 1, len(pips))
}

func Test_KeyUsedInPipelineJob(t *testing.T) {
	db, store := test.SetupPG(t, bootstrap.InitiliazeDB)
	if db == nil {
		t.FailNow()
	}
	u, _ := assets.InsertAdminUser(db)

	p := &sdk.Project{
		Key:  sdk.RandomString(10),
		Name: sdk.RandomString(10),
	}
	assert.NoError(t, project.Insert(db, store, p, u))

	k := sdk.ProjectKey{
		ProjectID: p.ID,
		Builtin:   false,
		Key: sdk.Key{
			Name: "proj-key",
		},
	}
	assert.NoError(t, project.InsertKey(db, &k))

	pip := &sdk.Pipeline{
		Name:      sdk.RandomString(10),
		ProjectID: p.ID,
		Type:      "build",
	}
	assert.NoError(t, pipeline.InsertPipeline(db, store, p, pip, u))

	s := &sdk.Stage{
		Name:       "Stage1",
		PipelineID: pip.ID,
	}
	assert.NoError(t, pipeline.InsertStage(db, s))

	j := &sdk.Job{
		PipelineStageID: s.ID,
		Action: sdk.Action{
			Name: "MyJOb",
		},
	}
	assert.NoError(t, pipeline.InsertJob(db, j, s.ID, pip))

	j.Action.Actions = []sdk.Action{
		{
			Name: sdk.ScriptAction,
			Parameters: []sdk.Parameter{
				{
					Name:  "key",
					Value: "proj-key",
				},
			},
		},
	}
	assert.NoError(t, pipeline.UpdateJob(db, j, u.ID))

	apps, pips := keyIsUsed(db, p.Key, "proj-key")
	assert.Equal(t, 0, len(apps))
	assert.Equal(t, 1, len(pips))
}
