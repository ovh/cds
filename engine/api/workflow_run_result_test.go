package api

import (
	"context"
	"fmt"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/bootstrap"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/engine/featureflipping"
	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
)

func Test_workflowRunArtifactCheckUpload(t *testing.T) {
	featureflipping.Init(gorpmapping.Mapper)
	api, db, router := newTestAPI(t, bootstrap.InitiliazeDB)

	u, _ := assets.InsertAdminUser(t, db)
	consumer, _ := authentication.LoadConsumerByTypeAndUserID(context.TODO(), db, sdk.ConsumerLocal, u.ID, authentication.LoadConsumerOptions.WithAuthentifiedUser)

	key := sdk.RandomString(10)
	proj := assets.InsertTestProject(t, db, api.Cache, key, key)

	feat := &sdk.Feature{
		Name: sdk.FeatureCDNArtifact,
		Rule: "return true",
	}
	require.NoError(t, featureflipping.Insert(gorpmapping.Mapper, db, feat))
	t.Cleanup(func() {
		_ = featureflipping.Delete(db, feat.ID)
	})

	w := assets.InsertTestWorkflow(t, db, api.Cache, proj, sdk.RandomString(10))

	wrCreate, err := workflow.CreateRun(api.mustDB(), w, sdk.WorkflowRunPostHandlerOption{AuthConsumerID: consumer.ID})
	assert.NoError(t, err)

	require.NoError(t, api.workflowRunCraft(context.TODO(), wrCreate.ID))

	wrDB, err := workflow.LoadRunByID(db, wrCreate.ID, workflow.LoadRunOptions{})
	require.NoError(t, err)

	nr := wrDB.WorkflowNodeRuns[w.WorkflowData.Node.ID][0]
	nr.Status = sdk.StatusBuilding
	require.NoError(t, workflow.UpdateNodeRun(db, &nr))

	nrj := nr.Stages[0].RunJobs[0]
	nrj.Status = sdk.StatusBuilding
	workflow.UpdateNodeJobRun(context.Background(), db, &nrj)

	cdnServices, _, jwtCDN := assets.InitCDNService(t, db)
	t.Cleanup(func() { _ = services.Delete(db, cdnServices) })

	//Prepare request
	vars := map[string]string{
		"key":              proj.Key,
		"permWorkflowName": w.Name,
		"number":           fmt.Sprintf("%d", wrCreate.Number),
		"nodeID":           fmt.Sprintf("%d", wrCreate.Workflow.WorkflowData.Node.ID),
	}
	checkRequest := sdk.CDNArtifactAPIRef{
		ArtifactName: "myArtifact",
		RunID:        wrCreate.ID,
		RunNodeID:    nr.ID,
		RunJobID:     nrj.ID,
		WorkflowID:   w.ID,
		WorkflowName: w.Name,
		ProjectKey:   key,
	}

	uri := router.GetRoute("POST", api.workflowRunArtifactCheckUpload, vars)
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtCDN, "POST", uri, checkRequest)

	//Do the request
	rec := httptest.NewRecorder()
	router.Mux.ServeHTTP(rec, req)
	assert.Equal(t, 204, rec.Code)
}
