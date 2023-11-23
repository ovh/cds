package api

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/ovh/cds/engine/api/authentication"
	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/engine/api/worker_v2"
	"github.com/ovh/cds/sdk/jws"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ovh/cds/engine/api/hatchery"
	"github.com/ovh/cds/engine/api/rbac"
	"github.com/ovh/cds/engine/api/region"
	"github.com/ovh/cds/engine/api/test/assets"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/sdk"
	sdkhatch "github.com/ovh/cds/sdk/hatchery"
	"github.com/rockbears/yaml"
	"github.com/stretchr/testify/require"
)

func TestWorkerUnregistered(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	reg := sdk.Region{Name: "default"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	rbacYaml := `name: perm-default
hatcheries:
- role: %s
  region: default
  hatchery: %s
`
	rbacYaml = fmt.Sprintf(rbacYaml, sdk.HatcheryRoleSpawn, hatch.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))
	r.Hatcheries[0].RegionID = reg.ID
	r.Hatcheries[0].HatcheryID = hatch.ID
	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	hatchConsumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusScheduling,
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
		HatcheryName:  hatch.Name,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

	workerName := sdk.RandomString(10)
	work, jwtWorker := assets.InsertWorker(t, ctx, db, hatchConsumer, hatch, workerName, jobRun)
	uri := api.Router.GetRouteV2("POST", api.postV2UnregisterWorkerHandler, map[string]string{"regionName": "default", "runJobID": jobRun.ID})
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtWorker, "POST", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)

	wkrDB, err := worker_v2.LoadByID(ctx, db, work.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusDisabled, wkrDB.Status)
}

func TestWorkerRefresh(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	reg := sdk.Region{Name: "default"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	rbacYaml := `name: perm-default
hatcheries:
- role: %s
  region: default
  hatchery: %s
`
	rbacYaml = fmt.Sprintf(rbacYaml, sdk.HatcheryRoleSpawn, hatch.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))
	r.Hatcheries[0].RegionID = reg.ID
	r.Hatcheries[0].HatcheryID = hatch.ID
	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	hatchConsumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusScheduling,
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
		HatcheryName:  hatch.Name,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

	workerName := sdk.RandomString(10)
	_, jwtWorker := assets.InsertWorker(t, ctx, db, hatchConsumer, hatch, workerName, jobRun)
	uri := api.Router.GetRouteV2("POST", api.postV2RefreshWorkerHandler, map[string]string{"regionName": "default", "runJobID": jobRun.ID})
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtWorker, "POST", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 204, w.Code)
}

func TestWorkerTakeJobHandler(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	reg := sdk.Region{Name: "default"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	rbacYaml := `name: perm-default
hatcheries:
- role: %s
  region: default
  hatchery: %s
`
	rbacYaml = fmt.Sprintf(rbacYaml, sdk.HatcheryRoleSpawn, hatch.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))
	r.Hatcheries[0].RegionID = reg.ID
	r.Hatcheries[0].HatcheryID = hatch.ID
	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	hatchConsumer, err := authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	jobRunSuccess := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusSuccess,
		JobID:         "myjob",
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
		HatcheryName:  hatch.Name,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRunSuccess))

	// Add run result
	rr := sdk.V2WorkflowRunResult{
		ID:               sdk.UUID(),
		WorkflowRunJobID: jobRunSuccess.ID,
		WorkflowRunID:    jobRunSuccess.WorkflowRunID,
		IssuedAt:         time.Now(),
		Status:           sdk.StatusSuccess,
		Type:             sdk.V2WorkflowRunResultTypeVariable,
		Detail: sdk.V2WorkflowRunResultDetail{
			Type: "V2WorkflowRunResultVariableDetail",
			Data: sdk.V2WorkflowRunResultVariableDetail{
				Name:  "foo",
				Value: "bar",
			},
		},
	}
	require.NoError(t, workflow_v2.InsertRunResult(ctx, db, &rr))

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusScheduling,
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
		HatcheryName:  hatch.Name,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

	workerName := sdk.RandomString(10)
	wkr, jwtWorker := assets.InsertWorker(t, ctx, db, hatchConsumer, hatch, workerName, jobRun)

	uri := api.Router.GetRouteV2("POST", api.postV2WorkerTakeJobHandler, map[string]string{"regionName": "default", "runJobID": jobRun.ID})
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, jwtWorker, "POST", uri, nil)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var takeJob sdk.V2TakeJobResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &takeJob))

	require.Equal(t, sdk.StatusBuilding, takeJob.RunJob.Status)
	require.Equal(t, workerName, takeJob.RunJob.WorkerName)

	wkDB, err := worker_v2.LoadByID(ctx, db, wkr.ID)
	require.NoError(t, err)
	require.Equal(t, sdk.StatusBuilding, wkDB.Status)

	require.Equal(t, 1, len(takeJob.Contexts.Jobs))
	jc, has := takeJob.Contexts.Jobs[jobRunSuccess.JobID]
	require.True(t, has)
	require.Equal(t, 1, len(jc.Outputs))
	require.Equal(t, "bar", jc.Outputs["foo"])
}

func TestWorkerRegister(t *testing.T) {
	api, db, _ := newTestAPI(t)
	ctx := context.TODO()

	db.Exec("DELETE FROM rbac")
	db.Exec("DELETE FROM region")

	admin, _ := assets.InsertAdminUser(t, db)
	proj := assets.InsertTestProject(t, db, api.Cache, sdk.RandomString(10), sdk.RandomString(10))
	vcsServer := assets.InsertTestVCSProject(t, db, proj.ID, "github", "github")
	repo := assets.InsertTestProjectRepository(t, db, proj.Key, vcsServer.ID, "myrepo")

	wkfName := sdk.RandomString(10)
	wr := sdk.V2WorkflowRun{
		Status:       sdk.StatusBuilding,
		ProjectKey:   proj.Key,
		UserID:       admin.ID,
		WorkflowName: wkfName,
		RepositoryID: repo.ID,
		VCSServerID:  vcsServer.ID,
	}
	require.NoError(t, workflow_v2.InsertRun(ctx, db, &wr))

	testPrivKey := "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEAyjvvBgiCxgsx/qt0jxUAZmD+vyEWJDe4oEGENGPzsvODoH9N\n6j23iXSO5NY2ZwiH2JteZwm760xhOJ2tb3fQ/louZMCOoA6h5IjqWUOYUHCeaZHj\nAOHmyfH1T51R1GUShzrPU4j6YqfaA2+z2D3iayZL2AHWJjFrz79qoDfa6dPrG0FC\ntspp+ZC/Tjokbff4BEZQ/WFDF5OdBHIEfCRZuJ5wW/isE+2WfR+h4UbnzlKCZHPI\ngZm85An+a3Mfqs+YN34qVoRi5SXNDt2axDaqkXpBACj9F/ZV12k1ZIrFhuPAdImA\nObYsJUy8f7MFnIzocIuIcckY0YfuFBPrdJvR/wIDAQABAoIBACfQIMBJUKV4csE2\nM94vPkLpeX1bICbFKX2zKDnAK6vhMNqZ9VbWC5nt7muymEc0cLn97FnQmsQ55zHk\nKM63AdfxHQ0Ms4djGhAJvEkSe5vqb+VMqSn1TyAWmDqyN/feZgVlWIeTgeeGL+9Z\nQEij9j0t7uR5iBVAyX6+qlkcZeZ+TdkHL+yS04EPAXrMB4YkZ3AI9wU6qkPogQzt\nMjzOe+GnkyUey/Kpch3+4Sg8fnwyPbP8Re7wkj7zhB/54d7lV+9fU47KVHZMtjAa\nvd6LKoltiAa3Yg66bmFAWsKTj8HAx14OxfKA7MJR+OSxRRs+sdA55st6q7DQyPNV\nwsLFTLECgYEA+tCSrFwmhG/+AnYRoVBhG9qlqZir7KE9NxQXnBNbjnKeKja1cnug\ncgKDq/Qv5ZKQJJjWYjiw9fIs/Tbhpvw0pVrfnSclE/tzBB/xb2T9L/KONUjcTM9M\nOe78DsRz3b0yWvx6K8Lh6CB+RPHQQj4vj9sDM57xlT80D2CnIZR7MUUCgYEAzmo/\n7hmBF6kIRTOjmtfTn8t9WqQhFtZ520Fh7LcGxDT8zj9BsiASUNEZncQuzf7zBUvL\nPGT0qKLHPJUUO4OQYFXJ3iM7NfqevCOQgBxqi98W4BlANmmr88EVzoJH1DKbMN8T\nfyN510zblAi0LavU0BfmPhjayNR4lQF4BGYE8HMCgYEAlegHYiEJnGpbRRlQXDvw\nbnd2QDFRwHJ2Ze8KVWx0cmUrB0v/ktc1BC9gh2vBHNNGd7kpmTcM/VKrdQRSRj3D\nMXlo4UK+NH07NyHoaY3QjdHJESvlD7tuccnWuPTN05/68sqoMnOeFeEl1ws4T/9e\n6foapcW/2lYWRYKdIcLfBokCgYAgi3YK93B4J4mLR6oK9u6B6laYXPpunGBYJoXn\nCbCCbQxTBHw6Nn5MybstOcpbZlAwzJN9sTk2AXxCXAr1mze/SKdsY8epKhuu9DiL\nSm+uH/1+Vdze92hKJW5nwfjLRzNa0EFFsXU7cf3I6FpoPQhvyuTdc5PDSGS+sZ2X\nq/IpawKBgQDr5uq1ljzVlLxmLarRqFI8EOxB1vDZt6C6M+J3BP+ukQH6AyzosNjU\no9wDxm08LOIueMoMe7PXN6tor96drHnVL2hNRanOke2rE309YesuYpTX6SzzLAxG\n42f1u+OhzpfypJJ+KlsWKpHpFi2iVq6NjwAyJZtP52I55y85pFK13A==\n-----END RSA PRIVATE KEY-----\n"
	pvtKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(testPrivKey))
	require.NoError(t, err)
	pubKey, err := jws.ExportPublicKey(pvtKey)
	require.NoError(t, err)

	hatch := sdk.Hatchery{
		ModelType: "docker",
		Name:      sdk.RandomString(10),
		PublicKey: pubKey,
	}
	require.NoError(t, hatchery.Insert(ctx, db, &hatch))

	reg := sdk.Region{Name: "default"}
	require.NoError(t, region.Insert(ctx, db, &reg))

	rbacYaml := `name: perm-default
hatcheries:
- role: %s
  region: default
  hatchery: %s
`
	rbacYaml = fmt.Sprintf(rbacYaml, sdk.HatcheryRoleSpawn, hatch.Name)
	var r sdk.RBAC
	require.NoError(t, yaml.Unmarshal([]byte(rbacYaml), &r))
	r.Hatcheries[0].RegionID = reg.ID
	r.Hatcheries[0].HatcheryID = hatch.ID
	require.NoError(t, rbac.Insert(context.TODO(), db, &r))

	_, err = authentication.NewConsumerHatchery(ctx, db, hatch)
	require.NoError(t, err)

	jobRun := sdk.V2WorkflowRunJob{
		ProjectKey:    proj.Key,
		UserID:        admin.ID,
		Status:        sdk.StatusScheduling,
		ModelType:     "docker",
		Region:        "default",
		WorkflowRunID: wr.ID,
		HatcheryName:  hatch.Name,
	}
	require.NoError(t, workflow_v2.InsertRunJob(ctx, db, &jobRun))

	registrationForm := sdk.WorkerRegistrationForm{
		OS:   "linux",
		Arch: "amd64",
	}
	spawn := sdkhatch.SpawnArguments{
		HatcheryName: hatch.Name,
		WorkerName:   sdk.RandomString(10),
		JobID:        jobRun.ID,
		WorkflowName: wr.WorkflowName,
		JobName:      jobRun.JobID,
	}

	workerToken, err := sdkhatch.NewWorkerTokenV2(hatch.Name, pvtKey, time.Now().Add(10*time.Minute), spawn)
	require.NoError(t, err)

	t.Logf("%+v", workerToken)

	uri := api.Router.GetRouteV2("POST", api.postV2RegisterWorkerHandler, map[string]string{"regionName": "default", "runJobID": jobRun.ID})
	test.NotEmpty(t, uri)
	req := assets.NewJWTAuthentifiedRequest(t, workerToken, "POST", uri, registrationForm)
	w := httptest.NewRecorder()
	api.Router.Mux.ServeHTTP(w, req)
	require.Equal(t, 200, w.Code)

	var work sdk.V2Worker
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &work))

	require.Equal(t, work.HatcheryName, hatch.Name)
	require.Equal(t, work.HatcheryID, hatch.ID)
}
