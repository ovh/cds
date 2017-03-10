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
	var1 := &sdk.Variable{
		Name:  "var1",
		Value: "value1",
		Type:  "PASSWORD",
	}
	err := project.InsertVariable(db, project1, var1, &sdk.User{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

	// 3. Test Update variable
	var1.Value = "value1Updated"
	err = project.UpdateVariable(db, project1, var1, &sdk.User{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot update var1 in project1: %s", err)
	}

	// 4. Delete variable
	err = project.DeleteVariable(db, project1, var1, &sdk.User{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot delete var1 from project: %s", err)
	}
	varTest, err := project.GetVariableInProject(db, project1.ID, var1.Name)
	if varTest.Value != "" {
		t.Fatalf("var1 should be deleted: %s", err)
	}

	// 5. Insert new var
	var2 := &sdk.Variable{
		Name:  "var2",
		Value: "value2",
		Type:  "STRING",
	}
	err = project.InsertVariable(db, project1, var2, &sdk.User{Username: "foo"})
	if err != nil {
		t.Fatalf("cannot insert var1 in project1: %s", err)
	}

}
