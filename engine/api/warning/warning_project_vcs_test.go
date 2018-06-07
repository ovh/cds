package warning

import (
	"fmt"
	"testing"

	"github.com/fatih/structs"
	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/sdk"
)

func TestUnusedProjectVCSWarning(t *testing.T) {
	db, cache := test.SetupPG(t, bootstrap.InitiliazeDB)
	u, _ := assets.InsertAdminUser(db)
	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, cache, key, key, u)

	// Create add vcs event
	ePayload := sdk.EventProjectVCSServerAdd{
		VCSServerName: "foo",
	}
	e := sdk.Event{
		ProjectKey: proj.Key,
		EventType:  fmt.Sprintf("%T", ePayload),
		Payload:    structs.Map(ePayload),
	}

	// Compute event
	warnToTest := unusedProjectVCSWarning{}
	test.NoError(t, warnToTest.compute(db, e))

	// Check warning exist
	warnsAfter, errAfter := GetByProject(db, proj.Key)
	test.NoError(t, errAfter)
	assert.Equal(t, 1, len(warnsAfter))

	(&warnsAfter[0]).ComputeMessage("en")
	t.Logf("%s", warnsAfter[0].Message)

	// Create Add key event
	ePayloadAdd := sdk.EventApplicationRepositoryAdd{
		VCSServer:  "foo",
		Repository: "ovh/cds",
	}
	eAdd := sdk.Event{
		ProjectKey:      proj.Key,
		ApplicationName: "foo",
		EventType:       fmt.Sprintf("%T", ePayloadAdd),
		Payload:         structs.Map(ePayloadAdd),
	}
	test.NoError(t, warnToTest.compute(db, eAdd))

	// Check that warning disapears
	warnsAdd, errAfterDelete := GetByProject(db, proj.Key)
	test.NoError(t, errAfterDelete)
	assert.Equal(t, 0, len(warnsAdd))

	// Deleting repo from application
	// Create Add key event
	ePayloadAppDelete := sdk.EventApplicationRepositoryDelete{
		VCSServer:  "foo",
		Repository: "ovh/cds",
	}
	eAppDelete := sdk.Event{
		ProjectKey:      proj.Key,
		ApplicationName: "foo",
		EventType:       fmt.Sprintf("%T", ePayloadAppDelete),
		Payload:         structs.Map(ePayloadAppDelete),
	}
	test.NoError(t, warnToTest.compute(db, eAppDelete))
	warnsAppDelete, errAppDeletz := GetByProject(db, proj.Key)
	test.NoError(t, errAppDeletz)
	assert.Equal(t, 1, len(warnsAppDelete))

	// Remove repo manager
	ePayloadRepoDelete := sdk.EventProjectVCSServerDelete{
		VCSServerName: "foo",
	}
	eAppDeleteRepo := sdk.Event{
		ProjectKey:      proj.Key,
		ApplicationName: "foo",
		EventType:       fmt.Sprintf("%T", ePayloadRepoDelete),
		Payload:         structs.Map(ePayloadRepoDelete),
	}
	test.NoError(t, warnToTest.compute(db, eAppDeleteRepo))
	warnsRepoDelete, errRepoDelete := GetByProject(db, proj.Key)
	test.NoError(t, errRepoDelete)
	assert.Equal(t, 0, len(warnsRepoDelete))
}
