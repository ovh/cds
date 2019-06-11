package internal_test

import (
	"context"
	"testing"
	"time"

	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/afero"
	"gopkg.in/h2non/gock.v1"
)

func TestStartWorker(t *testing.T) {
	defer gock.Off()

	gock.New("http://lolcat.host").Get("/action/requirement").
		Reply(200).
		JSON([]sdk.Requirement{
			{
				Name:  "bash",
				Type:  sdk.BinaryRequirement,
				Value: "bash",
			},
		})

	gock.New("http://lolcat.host").Post("/worker").
		Reply(201).
		JSON(sdk.Worker{
			ID:      "xxxx-xxxx-xxxxx",
			ModelID: 1,
		})

	gock.New("http://lolcat.host").Get("/worker/model").
		Reply(200).
		JSON([]sdk.Model{
			{
				ID:   1,
				Name: "my-worker-model",
			},
		})

	gock.New("http://lolcat.host").Get("/queue/workflows/42/infos").
		Reply(200).
		JSON(sdk.WorkflowNodeJobRun{
			ID: 42,
		})

	var w = new(internal.CurrentWorker)

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", 1, true, afero.NewMemMapFs()); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPSSEClient())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := internal.StartWorker(ctx, w, 42)
	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}
