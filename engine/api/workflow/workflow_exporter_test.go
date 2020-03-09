package workflow_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"testing"

	"github.com/ovh/cds/sdk/exportentities"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/environment"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
)

func TestPull(t *testing.T) {
	db, cache, end := test.SetupPG(t, bootstrap.InitiliazeDB)
	defer end()
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key)

	//First pipeline
	pip := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip1",
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip))

	s := sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip.ID
	pipeline.InsertStage(db, s)
	j := &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip)
	s.Jobs = append(s.Jobs, *j)

	pip.Stages = append(pip.Stages, *s)

	//Second pipeline
	pip2 := sdk.Pipeline{
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Name:       "pip2",
	}
	test.NoError(t, pipeline.InsertPipeline(db, &pip2))
	s = sdk.NewStage("stage 1")
	s.Enabled = true
	s.PipelineID = pip2.ID
	pipeline.InsertStage(db, s)
	j = &sdk.Job{
		Enabled: true,
		Action: sdk.Action{
			Enabled: true,
			Name:    "job20",
		},
	}
	pipeline.InsertJob(db, j, s.ID, &pip2)
	s.Jobs = append(s.Jobs, *j)

	//Application
	app := &sdk.Application{
		Name: sdk.RandomString(10),
	}
	test.NoError(t, application.Insert(db, cache, *proj, app))

	//Environment
	envName := sdk.RandomString(10)
	env := &sdk.Environment{
		ProjectID: proj.ID,
		Name:      envName,
	}
	test.NoError(t, environment.InsertEnvironment(db, env))

	w := sdk.Workflow{
		Name:       "test_1",
		ProjectID:  proj.ID,
		ProjectKey: proj.Key,
		Metadata:   sdk.Metadata{"triggered_by": "bla"},
		PurgeTags:  []string{"aa", "bb"},
		WorkflowData: &sdk.WorkflowData{
			Node: sdk.Node{
				Name: "node1",
				Ref:  "node1",
				Type: sdk.NodeTypePipeline,
				Context: &sdk.NodeContext{
					PipelineID: pip.ID,
				},
				Triggers: []sdk.NodeTrigger{
					{
						ChildNode: sdk.Node{
							Name: "node2",
							Ref:  "node2",
							Type: sdk.NodeTypePipeline,
							Context: &sdk.NodeContext{
								PipelineID:    pip2.ID,
								ApplicationID: app.ID,
								EnvironmentID: env.ID,
							},
						},
					},
				},
			},
		},
	}

	proj, _ = project.Load(db, cache, proj.Key, project.LoadOptions.WithApplications, project.LoadOptions.WithEnvironments, project.LoadOptions.WithPipelines)

	test.NoError(t, workflow.Insert(context.TODO(), db, cache, *proj, &w))

	w1, err := workflow.Load(context.TODO(), db, cache, *proj, "test_1", workflow.LoadOptions{})
	test.NoError(t, err)
	test.Equal(t, w.Metadata, w1.Metadata)
	test.Equal(t, w.PurgeTags, w1.PurgeTags)

	pull, err := workflow.Pull(context.TODO(), db, cache, *proj, w1.Name, exportentities.FormatYAML, project.EncryptWithBuiltinKey)
	test.NoError(t, err)

	buff := new(bytes.Buffer)
	test.NoError(t, pull.Tar(context.TODO(), buff))

	// Open the tar archive for reading.
	r := bytes.NewReader(buff.Bytes())
	tr := tar.NewReader(r)

	// Iterate through the files in the archive.
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		test.NoError(t, err, "Unable to iterate over the tar buffer")
		t.Logf("Contents of %s:", hdr.Name)

		btes, err := ioutil.ReadAll(tr)
		test.NoError(t, err, "Unable to read the tar buffer")

		t.Logf("%s", string(btes))

	}
}
