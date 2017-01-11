package main

import (
	"testing"

	_ "github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/pipeline"
	_ "github.com/ovh/cds/engine/api/project"
	_ "github.com/ovh/cds/engine/api/scheduler"
	"github.com/ovh/cds/engine/api/test"
	_ "github.com/ovh/cds/sdk"
)

func TestInsertLog(t *testing.T) {
	db := test.Setup("InsertLog", t)

	if err := pipeline.InsertLog(db, 3, "1", "hello world"); err != nil {
		t.Fatalf("Cannot insert log: %s", err)
	}
}

func TestLoadLogs(t *testing.T) {
	db := test.Setup("LoadLog", t)

	pipeline.InsertLog(db, 3, "1", "foo 2")
	pipeline.InsertLog(db, 3, "1", "foo 3")
	pipeline.InsertLog(db, 2, "1", "foo 1")
	pipeline.InsertLog(db, 2, "1", "foo 2")
	pipeline.InsertLog(db, 3, "1", "foo 4")

	pipeline.InsertLog(db, 3, "2", "foo 5")
	pipeline.InsertLog(db, 3, "2", "foo 6")
	pipeline.InsertLog(db, 3, "2", "foo 7")
	pipeline.InsertLog(db, 3, "2", "foo 8")

	logs, err := pipeline.LoadLogs(db, 3, 0, 0)
	if err != nil {
		t.Fatalf("Cannot load logs: %s", err)
	}

	if len(logs) != 7 {
		t.Fatalf("Expected 7 log lines, got %d", len(logs))
	}

	for _, l := range logs {
		if l.ActionBuildID != 3 {
			t.Fatalf("Got build id != 3 -> %d", l.ActionBuildID)
		}
	}
}

// FIXME ramsql subqueries
/*
func TestLoadPipelineBuildLogs(t *testing.T) {
	db := test.Setup("LoadPipelineBuildLogs", t)

	// 0- Create a project
	projectFoo := &sdk.Project{
		Name: "Foo",
		Key:  "FOO",
	}
	err := project.InsertProject(db, projectFoo)
	if err != nil {
		t.Fatalf("cannot insert project: %s", err)
	}

	// 1- Create a pipeline
	p := &sdk.Pipeline{
		Name:      "Foo",
		ProjectID: projectFoo.ID,
	}

	err = pipeline.InsertPipeline(db, p)
	if err != nil {
		t.Fatalf("cannot insert pipeline: %s", err)
	}

	// 2- Create some actions
	a := sdk.NewAction("foo")
	a.Step("echo", sdk.CommandStep, "echo 'bar space bar'")
	a.Requirement("foo", sdk.BinaryRequirement, "foo")

	err = action.InsertAction(db, a)
	if err != nil {
		t.Fatalf("Cannot insert action: %s\n", err)
	}

	b := sdk.NewAction("bar")
	b.Step("bar", sdk.CommandStep, "bar --cloud")
	b.Requirement("bar", sdk.BinaryRequirement, "bar")

	err = action.InsertAction(db, b)
	if err != nil {
		t.Fatalf("Cannot insert action: %s\n", err)
	}

	c := sdk.NewAction("git pull")
	c.Step("git pull", sdk.CommandStep, "git pull")
	c.Requirement("git", sdk.BinaryRequirement, "git")

	err = action.InsertAction(db, c)
	if err != nil {
		t.Fatalf("Cannot insert action: %s\n", err)
	}

	// 3- Add actions in pipeline
	err = pipeline.InsertPipelineAction(db, projectFoo.Key, p.Name, a.Name, "[]", 1)
	if err != nil {
		t.Fatalf("cannot add action %s to pipeline %s: %s", p.Name, a.Name, err)
	}
	err = pipeline.InsertPipelineAction(db, projectFoo.Key, p.Name, b.Name, "[]", 2)
	if err != nil {
		t.Fatalf("cannot add action %s to pipeline %s: %s", p.Name, b.Name, err)
	}
	err = pipeline.InsertPipelineAction(db, projectFoo.Key, p.Name, c.Name, "[]", 3)
	if err != nil {
		t.Fatalf("cannot add action %s to pipeline %s: %s", p.Name, c.Name, err)
	}

	// 4- Start a pipeline build
	pb, err := pipeline.InsertPipelineBuild(db, p, []string{})
	if err != nil {
		t.Fatalf("cannot insert pipeline build: %s\n", err)
	}

	// 5- Manually schedule build and insert logs
	buildData := &sdk.Build{
		PipelineActionID: 1,
		PipelineBuildID:  pb.ID,
	}

	err = scheduler.InsertBuild(db, buildData)
	if err != nil {
		t.Fatalf("cannot insert build: %s", err)
	}
	if buildData.ID == 0 {
		t.Fatalf("expected build id to bet not 0")
	}

	err = pipeline.InsertLog(db, buildData.ID, a.Name, "1")
	if err != nil {
		t.Fatalf("cannot insert log for build %d with action %s: %s", buildData.ID, a.Name, err)
	}

	buildData.PipelineActionID = 2
	buildData.ID = 0
	err = scheduler.InsertBuild(db, buildData)
	if err != nil {
		t.Fatalf("cannot insert build: %s", err)
	}
	if buildData.ID == 0 {
		t.Fatalf("expected build id to bet not 0")
	}

	err = pipeline.InsertLog(db, buildData.ID, b.Name, "2")
	if err != nil {
		t.Fatalf("cannot insert log for build %d with action %s: %s", buildData.ID, b.Name, err)
	}

	buildData.PipelineActionID = 3
	buildData.ID = 0
	err = scheduler.InsertBuild(db, buildData)
	if err != nil {
		t.Fatalf("cannot insert build: %s", err)
	}
	if buildData.ID == 0 {
		t.Fatalf("expected build id to bet not 0")
	}

	err = pipeline.InsertLog(db, buildData.ID, c.Name, "3")
	if err != nil {
		t.Fatalf("cannot insert log for build %d with action %s: %s", buildData.ID, c.Name, err)
	}

	// 6- Load logs
	logs, err := pipeline.LoadPipelineBuildLogs(db, pb.ID, 0)
	if err != nil {
		t.Fatalf("cannot load pipeline build logs: %s", err)
	}

	if len(logs) != 3 {
		t.Fatalf("expected 3 lines of log for this pipeline, got %d", len(logs))
	}

}
*/
