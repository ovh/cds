package action

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	_ "github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
)

func TestInsertAction(t *testing.T) {
	dba := test.Setup("InsertAction", t)
	db, err := dba.Begin()
	if err != nil {
		t.Fatalf("cannot start tx: %s\n", err)
	}

	script := sdk.NewAction(sdk.ScriptAction)
	script.Type = sdk.BuiltinAction
	script.Parameter(sdk.Parameter{Name: "script", Type: sdk.TextParameter})

	err = InsertAction(db, script, true)
	if err != nil {
		t.Fatalf("cannot insert script action: %s", err)
	}

	a := sdk.NewAction("foo")
	a.Add(sdk.NewScriptAction("echo 'bar space bar'"))
	a.Requirement("foo", sdk.BinaryRequirement, "foo")

	err = InsertAction(db, a, true)
	if err != nil {
		t.Fatalf("Cannot insert action: %s\n", err)
	}
}

func TestLoadAction(t *testing.T) {
	dba := test.Setup("LoadAction", t)
	db, err := dba.Begin()
	if err != nil {
		t.Fatalf("cannot start tx: %s\n", err)
	}

	bar := sdk.NewAction("bar")
	err = InsertAction(db, bar, true)
	if err != nil {
		t.Fatalf("Cannot insert action bar: %s", err)
	}

	a := sdk.NewAction("foo")
	a.Add(*bar)
	a.Requirement("foo", sdk.BinaryRequirement, "foo")

	err = InsertAction(db, a, true)
	if err != nil {
		t.Fatalf("Cannot insert action: %s", err)
	}

	a2, err := LoadPublicAction(db, "foo")
	if err != nil {
		t.Fatalf("Cannot load action foo: %s", err)
	}

	if a2.Name != "foo" {
		t.Fatalf("Expected action name to be 'foo', was '%s'", a2.Name)
	}

	if len(a2.Requirements) != 1 {
		t.Fatalf("Expected 1 requirements, got %d", len(a2.Requirements))
	}

	if len(a2.Actions) != 1 {
		t.Fatalf("Expected 1 step, got %d", len(a2.Actions))
	}
}
