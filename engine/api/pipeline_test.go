package main

import (
	"database/sql"
	"testing"

	"github.com/ovh/cds/engine/api/application"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/pipeline"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func insertTestPipeline(db *sql.DB, t *testing.T, name string) (*sdk.Project, *sdk.Pipeline, *sdk.Application) {

	projectFoo := &sdk.Project{
		Name: "Foo",
		Key:  "FOO",
	}
	err := project.InsertProject(db, projectFoo)
	if err != nil {
		t.Fatalf("cannot insert project: %s", err)
	}

	p := &sdk.Pipeline{
		Name:      name,
		ProjectID: projectFoo.ID,
	}

	app := &sdk.Application{
		Name: "App1",
	}

	err = application.InsertApplication(db, projectFoo, app)

	err = pipeline.InsertPipeline(db, p)
	if err != nil {
		t.Fatalf("cannot insert pipeline: %s", err)
	}

	err = application.AttachPipeline(db, app.ID, p.ID)

	return projectFoo, p, app
}

func TestInsertAndDeletePipeline(t *testing.T) {
	db := test.Setup("TestInsertAndDeletePipeline", t)

	p := &sdk.Pipeline{
		Name: "Foo",
	}

	err := pipeline.InsertPipeline(db, p)
	if err != nil {
		t.Fatalf("cannot insert pipeline: %s", err)
	}

	if p.ID == 0 {
		t.Fatal("expected id being not 0 after insert")
	}

	groupInsert := &sdk.Group{
		Name: "GroupeFoo",
	}

	err = group.InsertGroup(db, groupInsert)
	if err != nil {
		t.Fatalf("cannot insert group: %s", err)
	}
	if groupInsert.ID == 0 {
		t.Fatal("expected groupInsert.id being not 0 after insert")
	}

	err = group.InsertGroupInPipeline(db, p.ID, groupInsert.ID, 4)
	if err != nil {
		t.Fatalf("cannot add group in pipeline: %s", err)
	}

	// FIXME subqueries in ramsql
	/*
		err = pipeline.DeletePipeline(db, p.ID)
		if err != nil {
			t.Fatalf("cannot delete pipeline: %s", err)
		}

		query := `SELECT pipeline_id from pipeline_group`
		rows, err := db.Query(query)
		if rows.Next() || err != nil {
			var id int
			rows.Scan(&id)
			t.Fatalf("Should not have pipeline group for pipeline : %d", id)
		}

		query = `SELECT id from pipeline`
		rows, err = db.Query(query)
		if rows.Next() || err != nil {
			var id int
			rows.Scan(&id)
			t.Fatalf("Should not have pipeline for pipeline : %d", id)
		}
	*/

}

/*
func TestLoadPipelineHistory(t *testing.T) {
	db := test.Setup("LoadPipelineHistory", t)

	_, p := insertTestPipeline(db, t, "Foo")

	_, err := pipeline.LoadPipelineHistory(db, p.ID)
	if err != nil {
		t.Fatalf("cannot load pipeline history: %s\n", err)
	}
}
*/
// FIXME when ramsql will be able to do arithmetic operations
/*
func TestMoveStage(t *testing.T) {
	db := test.Setup("TestMoveStage", t)

	_, p := insertTestPipeline(db, t, "Foo")

	insertStage(db, t, p.ID, "Stage1", 1)
	s2 := insertStage(db, t, p.ID, "Stage2", 2)
	s3 := insertStage(db, t, p.ID, "Stage3", 3)
	insertStage(db, t, p.ID, "Stage4", 4)
	insertStage(db, t, p.ID, "Stage5", 5)

	err := pipeline.MoveStage(db, s3, 1)
	if err != nil {
		t.Fatalf("cannot move stage3 : %s\n", err)
	}

	err = pipeline.MoveStage(db, s2, 4)
	if err != nil {
		t.Fatalf("cannot move stage2 : %s\n", err)
	}

	s1Verif, _ := pipeline.LoadStage(db, p.ID, 1)
	if s1Verif.BuildOrder != 2 {
		t.Fatalf("Stage1 should be in 2nd position : %s\n", err)
	}
	s2Verif, _ := pipeline.LoadStage(db, p.ID, 2)
	if s2Verif.BuildOrder != 4 {
		t.Fatalf("Stage2 should be in 4th position : %s\n", err)
	}
	s3Verif, _ := pipeline.LoadStage(db, p.ID, 3)
	if s3Verif.BuildOrder != 1 {
		t.Fatalf("Stage3 should be in 1st position : %s\n", err)
	}
	s4Verif, _ := pipeline.LoadStage(db, p.ID, 4)
	if s4Verif.BuildOrder != 3 {
		t.Fatalf("Stage4 should be in 3rd position : %s\n", err)
	}
	s5Verif, _ := pipeline.LoadStage(db, p.ID, 5)
	if s5Verif.BuildOrder != 5 {
		t.Fatalf("Stage1 should be in 5th position : %s\n", err)
	}
}
*/
// FIXME ramsql subqueries
/*
func TestLoadPipelineActions(t *testing.T) {
	db := test.Setup("LoadPipelineActions", t)

	project, p := insertTestPipeline(db, t, "Foo")

	a := sdk.NewAction("foo")
	a.Step("echo", sdk.CommandStep, "echo 'bar space bar'")
	a.Requirement("foo", sdk.BinaryRequirement, "foo")

	err := action.InsertAction(db, a)
	if err != nil {
		t.Fatalf("Cannot insert action: %s\n", err)
	}

	err = pipeline.InsertPipelineAction(db, project.Key, p.Name, a.Name, "[{\"name\":\"yo\",\"value\":\"lol\"}]", 0)
	if err != nil {
		t.Fatalf("cannot add action into pipeline: %s", err)
	}

	reloaded, err := pipeline.LoadPipeline(db, project.Key, p.Name, true)
	if err != nil {
		t.Fatalf("Cannot load pipeline: %s", err)
	}

	if len(reloaded.Stages) != 1 {
		t.Fatalf("Expected 1 stage, got %d", len(reloaded.Stages))
	}

	if len(reloaded.Stages[0].Actions) != 1 {
		t.Fatalf("Expected 1 action, got %d", len(reloaded.Stages[0].Actions))
	}
}

func TestLoadBuildingPipelines(t *testing.T) {
	db := test.Setup("LoadBuildingPipelines", t)

	// 1- Create a pipeline
	project, p := insertTestPipeline(db, t, "Foo")

	// 2- Create some actions
	a := sdk.NewAction("foo")
	a.Step("echo", sdk.CommandStep, "echo 'bar space bar'")
	a.Requirement("foo", sdk.BinaryRequirement, "foo")

	err := action.InsertAction(db, a)
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
	err = pipeline.InsertPipelineAction(db, project.Key, p.Name, a.Name, "[]", 1)
	if err != nil {
		t.Fatalf("cannot add action %s to pipeline %s: %s", p.Name, a.Name, err)
	}
	err = pipeline.InsertPipelineAction(db, project.Key, p.Name, b.Name, "[]", 1)
	if err != nil {
		t.Fatalf("cannot add action %s to pipeline %s: %s", p.Name, b.Name, err)
	}
	err = pipeline.InsertPipelineAction(db, project.Key, p.Name, c.Name, "[]", 1)
	if err != nil {
		t.Fatalf("cannot add action %s to pipeline %s: %s", p.Name, c.Name, err)
	}

	// 4- Start a pipeline build
	_, err = pipeline.InsertPipelineBuild(db, p, []string{})
	if err != nil {
		t.Fatalf("cannot insert pipeline build: %s\n", err)
	}

	// 5- Load building pipelines
	pbs, err := pipeline.LoadBuildingPipelines(db)
	if err != nil {
		t.Fatalf("cannot load building pipelines: %s", err)
	}

	if len(pbs) != 1 {
		t.Fatalf("expected 1 building pipeline, got %d", len(pbs))
	}
}

func TestVariableInPipeline(t *testing.T) {

	db := test.Setup("TestVariableInPipeline", t)

	// 1. Create project and pipeline
	_, pipelineData := insertTestPipeline(db, t, "pipeline")

	// 2. Insert new variable
	var1 := sdk.Variable{
		Name:  "var1",
		Value: "value1",
		Type:  "PASSWORD",
	}
	err := pipeline.InsertVariableInPipeline(db, pipelineData.ID, var1)
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

	// 3. Test Update variable
	var1.Value = "value1Updated"
	err = pipeline.UpdateVariableInPipeline(db, pipelineData.ID, var1)
	if err != nil {
		t.Fatalf("cannot update var1 in project1: %s", err)
	}
	varTest, err := pipeline.GetVariableInPipeline(db, pipelineData.ID, var1.Name)
	if err != nil {
		t.Fatalf("cannot get var1 in project1: %s", err)
	}
	if varTest.Value != var1.Value {
		t.Fatalf("wrong value forvar1 in project1: %s", err)
	}

	// 4. Delete variable
	err = pipeline.DeleteVariableFromPipeline(db, pipelineData.ID, var1.Name)
	if err != nil {
		t.Fatalf("cannot delete var1 from project: %s", err)
	}
	varTest, err = pipeline.GetVariableInPipeline(db, pipelineData.ID, var1.Name)
	if varTest.Value != "" {
		t.Fatalf("var1 should be deleted: %s", err)
	}

	// 5. Insert new var
	var2 := sdk.Variable{
		Name:  "var2",
		Value: "value2",
		Type:  "STRING",
	}
	err = pipeline.InsertVariableInPipeline(db, pipelineData.ID, var2)
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

	// 6. Delete pipeline
	err = pipeline.DeletePipeline(db, pipelineData.ID)
	if err != nil {
		t.Fatalf("cannot delete project: %s", err)
	}
	varTest, err = pipeline.GetVariableInPipeline(db, pipelineData.ID, var2.Name)
	if err == nil || err != sql.ErrNoRows {
		t.Fatalf("var2 should be deleted: %s", err)
	}
}
*/
