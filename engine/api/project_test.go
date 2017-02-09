package main

import (
	"testing"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestVariableInProject(t *testing.T) {
	db := test.SetupPG(t, bootstrap.InitiliazeDB)

	// 1. Create project
	project1 := assets.InsertTestProject(t, db, assets.RandomString(t, 10), assets.RandomString(t, 10))

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
