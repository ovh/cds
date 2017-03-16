package main

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/loopfz/gadgeto/iffy"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/engine/api/project"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func Test_getVariableAuditInProjectHandler(t *testing.T) {
	db := test.SetupPG(t)

	router = &Router{auth.TestLocalAuth(t), mux.NewRouter(), "/Test_getVariableAuditInProjectHandler"}
	router.init()

	//Create admin user
	u, pass := assets.InsertAdminUser(t, db)

	//Create a fancy httptester
	tester := iffy.NewTester(t, router.mux)

	//Insert Project
	pkey := assets.RandomString(t, 10)
	proj := assets.InsertTestProject(t, db, pkey, pkey)

	// Add variable
	v := sdk.Variable{
		Name:  "foo",
		Type:  "string",
		Value: "bar",
	}
	if err := project.InsertVariable(db, proj, &v, u); err != nil {
		t.Fatal(err)
	}

	vars := map[string]string{
		"permProjectKey": proj.Key,
		"name":           "foo",
	}

	route := router.getRoute("GET", getVariableAuditInProjectHandler, vars)
	headers := assets.AuthHeaders(t, u, pass)

	var audits []sdk.ProjectVariableAudit
	tester.AddCall("Test_getVariableAuditInProjectHandler", "GET", route, nil).Headers(headers).Checkers(iffy.ExpectStatus(200), iffy.ExpectListLength(1), iffy.DumpResponse(t), iffy.UnmarshalResponse(&audits))
	tester.Run()

	assert.Nil(t, audits[0].VariableBefore)
	assert.Equal(t, audits[0].VariableAfter.Name, "foo")
}
