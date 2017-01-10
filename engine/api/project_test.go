package main

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/group"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

// TestInsertAndDelete test insert and delete project
func TestInsertAndDelete(t *testing.T) {
	db := test.Setup("TestInsertAndDelete", t)

	p := &sdk.Project{
		Key:  "KEY",
		Name: "name",
	}

	_ = project.InsertProject(db, p)

	if p.ID == 0 {
		t.Fatalf("ID cannot be 0 after insert")
	}

	groupInsert := &sdk.Group{
		Name: "GroupeFoo",
	}

	err := group.InsertGroup(db, groupInsert)
	if err != nil {
		t.Fatalf("cannot insert group: %s", err)
	}

	err = group.InsertGroupInProject(db, p.ID, groupInsert.ID, 4)
	if err != nil {
		t.Fatalf("cannot insert group in project: %s", err)
	}

}

func TestAddGroupInProject(t *testing.T) {

	db := test.Setup("TestAddGroupInProject", t)

	groupInsert := &sdk.Group{
		Name: "GroupeFoo",
	}

	err := group.InsertGroup(db, groupInsert)
	if err != nil {
		t.Fatalf("cannot insert group: %s", err)
	}
	if groupInsert.ID == 0 {
		t.Fatalf("groupInsert.ID cannot be 0")
	}

	g, err := group.LoadGroup(db, "GroupeFoo")
	if err != nil {
		t.Fatalf("cannot load group: %s", err)
	}
	if g.ID == 0 {
		t.Fatalf("g.ID cannot be 0")
	}

	project1 := &sdk.Project{
		Key:  "foo",
		Name: "foo",
	}
	project2 := &sdk.Project{
		Key:  "bar",
		Name: "bar",
	}

	err = project.InsertProject(db, project1)
	if err != nil {
		t.Fatalf("cannot insert project1: %s", err)
	}
	err = project.InsertProject(db, project2)
	if err != nil {
		t.Fatalf("cannot insert project2: %s", err)
	}

	err = group.InsertGroupInProject(db, project1.ID, g.ID, 4)
	if err != nil {
		t.Fatalf("cannot insert project1 in group: %s", err)
	}
	err = group.InsertGroupInProject(db, project2.ID, g.ID, 5)
	if err != nil {
		t.Fatalf("cannot insert project1 in group: %s", err)
	}

	err = project.LoadProjectByGroup(db, g)
	if err != nil {
		t.Fatalf("cannot load project by group: %s", err)
	}

	if len(g.ProjectGroups) != 2 {
		t.Fatalf("Wrong number of user, should be 2, got %d", len(g.ProjectGroups))
	}
	if g.ProjectGroups[0].Project.Key != "bar" {
		t.Fatalf("Wrong project, should be bar, got %s", g.ProjectGroups[0].Project.Key)
	}
	if g.ProjectGroups[0].Permission != 5 {
		t.Fatalf("Wrong role on project, should be 5, got %d", g.ProjectGroups[0].Permission)
	}
	if g.ProjectGroups[1].Project.Key != "foo" {
		t.Fatalf("Wrong project, should be foo, got %s", g.ProjectGroups[1].Project.Key)
	}
	if g.ProjectGroups[1].Permission != 4 {
		t.Fatalf("Wrong role on project, should be 4, got %d", g.ProjectGroups[1].Permission)
	}
}

func TestVariableInProject(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	// 1. Create project
	project1 := test.InsertTestProject(t, db, test.RandomString(t, 10), test.RandomString(t, 10))

	// 2. Insert new variable
	var1 := sdk.Variable{
		Name:  "var1",
		Value: "value1",
		Type:  "PASSWORD",
	}
	err := project.InsertVariableInProject(db, project1, var1)
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

	// 3. Test Update variable
	var1.Value = "value1Updated"
	err = project.UpdateVariableInProject(db, project1, var1)
	if err != nil {
		t.Fatalf("cannot update var1 in project1: %s", err)
	}
	/* ramsql doesn't handle bytes array, must be tested in it
	varTest, err := project.GetVariableInProject(db, project1.ID, var1.Name)
	if err != nil {
		t.Fatalf("cannot get var1 in project1: %s", err)
	}
	if varTest.Value != var1.Value {
		t.Fatalf("wrong value forvar1 in project1, expected '%s', got '%s'", var1.Value, varTest.Value)
	}
	*/

	// 4. Delete variable
	err = project.DeleteVariableFromProject(db, project1, var1.Name)
	if err != nil {
		t.Fatalf("cannot delete var1 from project: %s", err)
	}
	varTest, err := project.GetVariableInProject(db, project1.ID, var1.Name)
	if varTest.Value != "" {
		t.Fatalf("var1 should be deleted: %s", err)
	}

	// 5. Insert new var
	var2 := sdk.Variable{
		Name:  "var2",
		Value: "value2",
		Type:  "STRING",
	}
	err = project.InsertVariableInProject(db, project1, var2)
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

	// FIXME subqueries in ramsql
	/*
		// 6. Delete project
		err = project.DeleteProject(db, project1.Key)
		if err != nil {
			t.Fatalf("cannot delete project: %s", err)
		}
		varTest, err = project.GetVariableInProject(db, project1.ID, var2.Name)
		if err == nil || err != sql.ErrNoRows {
			t.Fatalf("var2 should be deleted: %s", err)
		}
	*/
}
