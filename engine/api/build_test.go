package main

import (
	"testing"

	"github.com/ovh/cds/engine/api/build"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/scheduler"
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

	err = scheduler.InsertBuild(db, b)
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

	err = scheduler.InsertBuild(db, b)
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

// FIXME ramsql subqueries
/*
func TestGetBuildResult(t *testing.T) {
	db := test.Setup("TestGetBuildResult", t)

	_, p := insertTestPipeline(db, t, "Foo")

	actionData := &sdk.Action{
		Name: "Action1",
	}
	err := action.InsertAction(db, actionData)
	if err != nil {
		t.Fatalf("Cannot insert action1: %s", err)
	}

	actionData = &sdk.Action{
		Name: "Action2",
	}
	err = action.InsertAction(db, actionData)
	if err != nil {
		t.Fatalf("Cannot insert action2: %s", err)
	}

	err = pipeline.InsertPipelineAction(db, "FOO", "Foo", "Action1", "[{\"name\":\"yo\",\"value\":\"lol\"}]", 0)
	if err != nil {
		t.Fatalf("cannot insert action: %s", err)
	}

	err = pipeline.InsertPipelineAction(db, "FOO", "Foo", "Action2", "[{\"name\":\"yo\",\"value\":\"lol\"}]", 0)
	if err != nil {
		t.Fatalf("cannot insert action: %s", err)
	}

	pb, err := pipeline.InsertPipelineBuild(db, p, []string{})
	if err != nil {
		t.Fatalf("cannot insert pipeline build: %s\n", err)
	}

	b := &sdk.Build{
		ID:               0,
		PipelineActionID: 1,
		PipelineBuildID:  pb.ID,
		Status:           sdk.StatusSuccess,
	}
	err = scheduler.InsertBuild(db, b)
	if err != nil {
		t.Fatalf("cannot insert build: %s", err)
	}
	if b.ID == 0 {
		t.Fatalf("expected build id to be not 0")
	}

	b = &sdk.Build{
		ID:               0,
		PipelineActionID: 2,
		PipelineBuildID:  pb.ID,
		Status:           sdk.StatusFail,
	}
	err = scheduler.InsertBuild(db, b)
	if err != nil {
		t.Fatalf("cannot insert build: %s", err)
	}
	if b.ID == 0 {
		t.Fatalf("expected build id to be not 0")
	}

	actionsBuild, err := build.LoadBuildByPipelineBuildID(db, pb.ID)
	if err != nil {
		t.Fatalf("cannot load builds: %s", err)
	}
	if len(actionsBuild) != 2 {
		t.Fatalf("Should have 2 actions builds, goet %d", len(actionsBuild))
	}
	for _, build := range actionsBuild {
		if build.PipelineStageID == 0 {
			t.Fatalf("Build should be attach to a stage")
		}
	}

}
*/
