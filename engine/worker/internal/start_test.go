package internal_test

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/afero"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/engine/worker/internal"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/log"
	"github.com/stretchr/testify/assert"
)

func init() {
	log.Initialize(&log.Conf{Level: "debug"})
}

func TestStartWorkerWithABookedJob(t *testing.T) {
	defer gock.Off()
	gock.Observe(gock.DumpRequest)

	gock.New("http://lolcat.host").Get("/action/requirement").
		Reply(200).
		JSON([]sdk.Requirement{
			{
				Name:  "bash",
				Type:  sdk.BinaryRequirement,
				Value: "bash",
			},
		})

	gock.New("http://lolcat.host").Post("/auth/consumer/worker/signin").
		HeaderPresent("Authorization").
		Reply(201).
		JSON(sdk.Worker{
			ID:      "xxxx-xxxx-xxxxx",
			ModelID: 1,
		}).AddHeader("X-CDS-JWT", "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJJRCI6IjJiNTA3ZDZiLTlhZWYtNGNlNS04MzhlLTA1OTU5NjhjMGU5NSIsIkdyb3VwSURzIjpbMV0sIlNjb3BlcyI6WyJXb3JrZXIiLCJSdW5FeGVjdXRpb24iXSwiZXhwIjoxNTYwNTA5NTQyLCJqdGkiOiIyYjUwN2Q2Yi05YWVmLTRjZTUtODM4ZS0wNTk1OTY4YzBlOTUiLCJpYXQiOjE1NjA1MDU5NDIsImlzcyI6ImNkc190ZXN0Iiwic3ViIjoiMTU4ODY5M2YtOTE5NC00ODg5LWJmYjAtZWY3Nzc5M2QzY2ViIn0.jLot6mtYHdnNAKxUS7OK7d6fVyMQyc7fS2NW4s727dxjx01Q2pPUQJBr16gKsS4ETSKh2ik7kqGGXdOz3i67DxMlPHcs0Azka1VOlefPcA77is-oVu0MPh4JbL0KA7fCu_98VKLJH3B0jYr4HEG9285ZOjFg7L5yuR7OqeFfCE3MrigyMKaNOrNE2FohOK9o50GyW_pAr6uNXcTu-yvqQUsz2B2gsd90HK2iWnvb8pKnBVVPg9Q0VA5l2IoFZR_p_UKSJZcFyFnjWKBVy33b70xupDnCpD-3OcbIFAQ5NPRvU_BnEjj_Jm59Ljrv3pJt1ErTMTkMA9QIFdYkDp5a6Q")

	gock.New("http://lolcat.host").Get("/worker/model").
		HeaderPresent("Authorization").
		Reply(200).
		JSON([]sdk.Model{
			{
				ID:   1,
				Name: "my-worker-model",
			},
		})

	gock.New("http://lolcat.host").Post("/worker/waiting").Times(2).
		HeaderPresent("Authorization").
		Reply(200).JSON(nil)

	gock.New("http://lolcat.host").Get("/queue/workflows/42/infos").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(sdk.WorkflowNodeJobRun{
			ID: 42,
		})

	gock.New("http://lolcat.host").Post("/queue/workflows/42/take").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(
			sdk.WorkflowNodeJobRunData{
				NodeJobRun: sdk.WorkflowNodeJobRun{
					ID: 42,
					Parameters: []sdk.Parameter{
						{
							Name:  "cds.project",
							Value: "my-project",
						},
						{
							Name:  "cds.workflow",
							Value: "my-workflow",
						},
						{
							Name:  "cds.node",
							Value: "my-node",
						},
						{
							Name:  "cds.job",
							Value: "my-job",
						},
					},
					Job: sdk.ExecutedJob{
						Job: sdk.Job{
							Enabled: true,
							Action: sdk.Action{
								Name:    "First Job",
								Enabled: true,
								Actions: []sdk.Action{
									{
										Name:     sdk.CheckoutApplicationAction,
										Type:     sdk.BuiltinAction,
										Enabled:  true,
										StepName: "checkout",
										Parameters: []sdk.Parameter{
											{
												Name:  "directory",
												Value: "{{.cds.workspace}}",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		)

	gock.New("http://lolcat.host").Post("/queue/workflows/42/step").Times(2).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://lolcat.host").Post("/queue/workflows/42/log").Times(1).
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://lolcat.host").Post("/queue/workflows/42/result").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	gock.New("http://lolcat.host").Post("/auth/consumer/worker/signout").
		HeaderPresent("Authorization").
		Reply(200).
		JSON(nil)

	var w = new(internal.CurrentWorker)

	if err := w.Init("test-worker", "test-hatchery", "http://lolcat.host", "xxx-my-token", 1, true, afero.NewMemMapFs()); err != nil {
		t.Fatalf("worker init failed: %v", err)
	}
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(w.Client().(cdsclient.Raw).HTTPSSEClient())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := internal.StartWorker(ctx, w, 42)
	assert.NoError(t, err)
	assert.True(t, gock.IsDone())
}
