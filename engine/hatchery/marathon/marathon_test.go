package marathon

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gambol99/go-marathon"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/hatchery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

func TestWorkerStarted(t *testing.T) {
	defer gock.Off()
	h := InitMarathonMarathonTest(marathonJDD{
		DeploymentTime:     5,
		WorkerSpawnTimeout: 5,
		Prefix:             "/my/workers",
	})

	apps := marathon.Applications{
		Apps: []marathon.Application{
			{
				ID: "/my/workers/w1",
			},
			{
				ID: "/my/workers/w2",
			},
		},
	}

	gock.New("http://mara.thon").Get("/v2/apps").Reply(200).JSON(apps)
	wkrs, err := h.WorkersStarted(context.TODO())
  require.NoError(t, err)
	t.Logf("%+v", wkrs)
	assert.Equal(t, 2, len(wkrs))
	assert.Equal(t, "w1", wkrs[0])
	assert.Equal(t, "w2", wkrs[1])
	assert.True(t, gock.IsDone())
}

func TestKillDisabledWorker(t *testing.T) {
	defer gock.Off()
	h := InitMarathonMarathonTest(marathonJDD{
		DeploymentTime:     5,
		WorkerSpawnTimeout: 5,
		Prefix:             "/my/workers",
	})

	workers := []sdk.Worker{
		{
			Name:   "model1-toto",
			Status: sdk.StatusDisabled,
		},
		{
			Name:   "model1-toto2",
			Status: sdk.StatusSuccess,
		},
		{
			Name:   "model2-toto",
			Status: sdk.StatusBuilding,
		},
		{
			Name:   "model2-toto2",
			Status: sdk.StatusDisabled,
		},
	}
	gock.New("http://lolcat.host").Get("/worker").Reply(200).JSON(workers)

	apps := marathon.Applications{
		Apps: []marathon.Application{
			{
				ID: "/my/workers/model1-toto",
			},
			{
				ID: "/my/workers/model1-toto2",
			},
			{
				ID: "/my/workers/model2-toto",
			},
			{
				ID: "/my/workers/model2-toto2",
			},
			{
				ID: "/my/workers/model3-toto",
			},
		},
	}
	gock.New("http://mara.thon").Get("/v2/apps").Reply(200).JSON(apps)

	respDelete := marathon.DeploymentID{}
	gock.New("http://mara.thon").Delete("/v2/apps/my/workers/model1-toto").Reply(200).JSON(respDelete)
	gock.New("http://mara.thon").Delete("/v2/apps/my/workers/model2-toto2").Reply(200).JSON(respDelete)

	err := h.killDisabledWorkers()
	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}

func TestKillAwolWOrkers(t *testing.T) {
	defer gock.Off()
	h := InitMarathonMarathonTest(marathonJDD{
		DeploymentTime:     5,
		WorkerSpawnTimeout: 5,
		Prefix:             "/my/workers",
	})

	workers := []sdk.Worker{
		{
			Name:   "model1-toto",
			Status: sdk.StatusBuilding,
		},
		{
			Name:   "model1-toto2",
			Status: sdk.StatusBuilding,
		},
		{
			Name:   "model1-toto3",
			Status: sdk.StatusDisabled,
		},
		{
			Name:   "model2-toto",
			Status: sdk.StatusBuilding,
		},
		{
			Name:   "model2-toto2",
			Status: sdk.StatusBuilding,
		},
		{
			Name:   "model2-toto3",
			Status: sdk.StatusDisabled,
		},
	}
	gock.New("http://lolcat.host").Get("/worker").Reply(200).JSON(workers)

	d := time.Now()
	d3 := d.Add(-3 * time.Minute)
	d6 := d.Add(-6 * time.Minute)
	apps := marathon.Applications{
		Apps: []marathon.Application{
			{
				ID:      "/my/workers/model1-toto",
				Version: d.Format(time.RFC3339),
			},
			{
				ID:      "/my/workers/model1-toto2",
				Version: d3.Format(time.RFC3339),
			},
			{
				ID:      "/my/workers/model1-toto3",
				Version: d3.Format(time.RFC3339),
			},
			{
				ID:      "/my/workers/register-model2-toto",
				Version: d.Format(time.RFC3339),
			},
			{
				ID:      "/my/workers/register-model2-toto2",
				Version: d6.Format(time.RFC3339),
			},
			{
				ID:      "/my/workers/register-model2-toto3",
				Version: d6.Format(time.RFC3339),
			},
		},
	}
	gock.New("http://mara.thon").Get("/v2/apps").Reply(200).JSON(apps)

	respDelete := marathon.DeploymentID{}
	gock.New("http://mara.thon").Delete("/v2/apps/my/workers/model1-toto3").Reply(200).JSON(respDelete)
	gock.New("http://mara.thon").Delete("/v2/apps/my/workers/register-model2-toto3").Reply(200).JSON(respDelete)

	err := h.killAwolWorkers()
	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}

func TestSpawn(t *testing.T) {
	defer gock.Off()
	h := InitMarathonMarathonTest(marathonJDD{
		DeploymentTime:     5,
		WorkerSpawnTimeout: 5,
	})
	m := &sdk.Model{
		Name: "fake",
		Group: &sdk.Group{
			ID:   1,
			Name: "GroupModel",
		},
	}
	createAppResult := marathon.Application{
		ID: "",
		Deployments: []map[string]string{
			{
				"id": "aaa",
			},
		},
	}
	getAppResult := struct {
		Application *marathon.Application `json:"app"`
	}{&createAppResult}

	gock.New("http://mara.thon").Post("/v2/apps").Reply(200).JSON(createAppResult)
	gock.New("http://mara.thon").Get("/v2/deployments").Reply(200).JSON([]*marathon.Deployment{})
	gock.New("http://mara.thon").AddMatcher(func(r *http.Request, rr *gock.Request) (bool, error) {
		b, err := gock.MatchPath(r, rr)
		assert.NoError(t, err)
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.String(), "http://mara.thon/v2/apps/fake-") {
			if b {
				return true, nil
			}
			return false, nil
		}
		return true, nil
	}).Reply(200).JSON(getAppResult)

	jobID := 1
	err := h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{
		JobID:        int64(jobID),
		Model:        m,
		RegisterOnly: false,
		Requirements: nil,
	})
	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}

func TestSpawnWorkerTimeout(t *testing.T) {
	defer gock.Off()
	h := InitMarathonMarathonTest(marathonJDD{
		DeploymentTime:     5,
		WorkerSpawnTimeout: 1,
	})
	m := &sdk.Model{
		Name: "fake",
		Group: &sdk.Group{
			ID:   1,
			Name: "GroupModel",
		},
	}
	createAppResult := marathon.Application{
		ID: "",
		Deployments: []map[string]string{
			{
				"id": "aaa",
			},
		},
	}
	getAppResult := struct {
		Application *marathon.Application `json:"app"`
	}{&createAppResult}

	depID := marathon.DeploymentID{
		DeploymentID: "aaa",
		Version:      "1",
	}

	deps := []*marathon.Deployment{
		{
			ID: "aaa",
		},
	}

	gock.New("http://mara.thon").Post("/v2/apps").Reply(200).JSON(createAppResult)
	gock.New("http://mara.thon").Delete("/v2/deployments/aaa").Reply(200).JSON(depID)
	gock.New("http://mara.thon").Get("/v2/deployments").Persist().Reply(200).JSON(deps)
	gock.New("http://mara.thon").AddMatcher(func(r *http.Request, rr *gock.Request) (bool, error) {
		b, err := gock.MatchPath(r, rr)
		assert.NoError(t, err)
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.String(), "http://mara.thon/v2/apps/fake-") {
			if b {
				return true, nil
			}
			return false, nil
		}
		return true, nil
	}).Reply(200).JSON(getAppResult)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		if request.Body == nil {
			return
		}
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			switch {
			case request.Method == http.MethodPost && request.URL.String() == "http://mara.thon/v2/apps":
				var a marathon.Application
				assert.NoError(t, json.Unmarshal(bodyContent, &a))
				assert.Equal(t, "DOCKER", a.Container.Type)
				assert.Equal(t, "BRIDGE", a.Container.Docker.Network)
				assert.Equal(t, float64(1), a.CPUs)
				assert.Equal(t, 1, *a.Instances)
				assert.NotEmpty(t, (*a.Env)["CDS_CONFIG"])

				createAppResult.ID = a.ID
				createAppResult.Env = a.Env
				createAppResult.CPUs = a.CPUs
				createAppResult.Instances = a.Instances
				createAppResult.Container = a.Container
			}
		}
	}
	gock.Observe(checkRequest)

	jobID := 1
	err := h.SpawnWorker(context.TODO(), hatchery.SpawnArguments{
		JobID:        int64(jobID),
		Model:        m,
		RegisterOnly: false,
		Requirements: nil,
	})
	t.Logf("%+v\n", err)
	assert.Error(t, err)
	assert.Equal(t, "internal server error (caused by: deployment for aaa timeout)", err.Error())
}

func TestCanSpawn(t *testing.T) {
	defer gock.Off()
	h := InitMarathonMarathonTest(marathonJDD{
		MaxWorker:    1,
		MaxProvision: 1,
	})
	gock.New("http://mara.thon").Get("/v2/deployments").Reply(200).JSON([]*marathon.DeploymentID{})
	gock.New("http://mara.thon").Get("/v2/apps").Reply(200).JSON(marathon.Applications{})
	m := &sdk.Model{Name: "fake"}
	jobID := 1
	assert.True(t, h.CanSpawn(context.TODO(), m, int64(jobID), nil))
	assert.True(t, gock.IsDone())
}

func TestCanSpawnMaxProvisionReached(t *testing.T) {
	defer gock.Off()
	deps := []*marathon.Deployment{
		{
			ID: "deploy1",
		},
	}
	jdd := marathonJDD{
		MaxProvision: 1,
	}
	h := InitMarathonMarathonTest(jdd)
	gock.New("http://mara.thon").Get("/v2/deployments").Reply(200).JSON(deps)

	m := &sdk.Model{Name: "fake"}
	canSpawn := h.CanSpawn(context.TODO(), m, int64(1), nil)
	assert.False(t, canSpawn)
	assert.True(t, gock.IsDone())
}

func TestCanSpawnMaxContainerReached(t *testing.T) {
	defer gock.Off()
	jdd := marathonJDD{
		MaxWorker: 5,
	}
	apps := marathon.Applications{
		Apps: []marathon.Application{
			{
				ID: "app1",
			},
			{
				ID: "app2",
			},
			{
				ID: "app3",
			},
			{
				ID: "app4",
			},
			{
				ID: "app5",
			},
		},
	}

	h := InitMarathonMarathonTest(jdd)
	gock.New("http://mara.thon").Get("/v2/deployments").Reply(200).JSON([]*marathon.DeploymentID{})
	gock.New("http://mara.thon").Get("/v2/apps").Reply(200).JSON(apps)

	m := &sdk.Model{Name: "fake"}
	canSpawn := h.CanSpawn(context.TODO(), m, int64(1), nil)
	assert.False(t, canSpawn)
	assert.True(t, gock.IsDone())
}

func TestCanSpawnWithService(t *testing.T) {
	h := InitMarathonMarathonTest(marathonJDD{})
	m := &sdk.Model{Name: "fake"}
	canSpawn := h.CanSpawn(context.TODO(), m, int64(1), []sdk.Requirement{{Name: "pg", Type: sdk.ServiceRequirement, Value: "postgres:9.5.4"}})
	assert.False(t, canSpawn)
}
