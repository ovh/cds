package main

import (
	"testing"

	"github.com/ovh/cds/engine/api/build"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/queue"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestInsertBuild(t *testing.T) {
	db := test.Setup("InsertBuild", t)

	project, p, app := insertTestPipeline(db, t, "Foo")

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}

	trigger := sdk.PipelineBuildTrigger{
		ManualTrigger: true,
	}
	pb, err := pipeline.InsertPipelineBuild(tx, project, p, app, []sdk.Parameter{}, []sdk.Parameter{}, &sdk.DefaultEnv, 0, trigger)
	if err != nil {
		t.Fatalf("cannot insert pipeline build: %s\n", err)
	}

	b := &sdk.ActionBuild{
		PipelineActionID: 1,
		PipelineBuildID:  pb.ID,
	}

	err = queue.InsertActionBuild(db, b)
	if err != nil {
		t.Fatalf("cannot insert build: %s", err)
	}

	if b.ID == 0 {
		t.Fatalf("expected build id to be not 0")
	}
}

func TestUpdateActionBuildStatus(t *testing.T) {
	db := test.Setup("UpdateActionBuildStatus", t)

	project, p, app := insertTestPipeline(db, t, "Foo")

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Cannot begin tx: %s", err)
	}

	trigger := sdk.PipelineBuildTrigger{
		ManualTrigger: true,
	}
	pb, err := pipeline.InsertPipelineBuild(tx, project, p, app, []sdk.Parameter{}, []sdk.Parameter{}, &sdk.DefaultEnv, 0, trigger)
	if err != nil {
		t.Fatalf("cannot insert pipeline build: %s\n", err)
	}

	b := &sdk.ActionBuild{
		PipelineActionID: 1,
		PipelineBuildID:  pb.ID,
	}

	err = queue.InsertActionBuild(db, b)
	if err != nil {
		t.Fatalf("cannot insert build: %s", err)
	}

	err = build.UpdateActionBuildStatus(tx, b, sdk.StatusBuilding)
	if err != nil {
		t.Fatalf("cannot update action build status: %s", err)
	}

	if err = tx.Commit(); err != nil {
		t.Fatalf("cannot commit tx: %s", err)
	}
}
