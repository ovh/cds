package action

import (
	"bytes"
	"encoding/json"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"testing"

	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestRunRelease(t *testing.T) {
	defer gock.Off()

	wk, ctx := setupTest(t)

	gock.New("http://lolcat.host").Post("/project/projKey/workflows/workflowName/runs/999/nodes/666/release").
		Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := ioutil.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = ioutil.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
			switch mock.Request().URLStruct.String() {
			case "http://lolcat.host/queue/workflows/666/coverage":
				var releaseRequest sdk.WorkflowNodeRunRelease
				err := json.Unmarshal(bodyContent, &releaseRequest)
				assert.NoError(t, err)
				require.Equal(t, "1.1.1", releaseRequest.TagName)
				require.Equal(t, "My Title", releaseRequest.ReleaseTitle)
				require.Equal(t, "My description", releaseRequest.ReleaseContent)
				require.Equal(t, []string{"*.deb"}, releaseRequest.Artifacts)
			}
		}
	}
	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPSSEClient())
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "cds.project",
			Value: "projKey",
		},
		{
			Name:  "cds.workflow",
			Value: "workflowName",
		},
		{
			Name:  "cds.run.number",
			Value: "999",
		},
	}...)
	res, err := RunRelease(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "artifacts",
					Value: "*.deb",
				},
				{
					Name:  "tag",
					Value: "1.1.1",
				},
				{
					Name:  "title",
					Value: "My Title",
				},
				{
					Name:  "releaseNote",
					Value: "My description",
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
	assert.True(t, gock.IsDone())
}

func TestRunReleaseMissingTag(t *testing.T) {
	defer gock.Off()

	wk, ctx := setupTest(t)
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "cds.project",
			Value: "projKey",
		},
		{
			Name:  "cds.workflow",
			Value: "workflowName",
		},
		{
			Name:  "cds.run.number",
			Value: "999",
		},
	}...)
	res, err := RunRelease(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "artifacts",
					Value: "*.deb",
				},
				{
					Name:  "title",
					Value: "My Title",
				},
				{
					Name:  "releaseNote",
					Value: "My description",
				},
			},
		}, nil)
	assert.Error(t, err)
	assert.Contains(t, "tag name is not set. Nothing to perform", err.Error())
	assert.Equal(t, sdk.StatusFail, res.Status)
}

func TestRunReleaseMissingTitle(t *testing.T) {
	defer gock.Off()

	wk, ctx := setupTest(t)
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "cds.project",
			Value: "projKey",
		},
		{
			Name:  "cds.workflow",
			Value: "workflowName",
		},
		{
			Name:  "cds.run.number",
			Value: "999",
		},
	}...)
	res, err := RunRelease(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "artifacts",
					Value: "*.deb",
				},
				{
					Name:  "tag",
					Value: "1.1.1",
				},
				{
					Name:  "releaseNote",
					Value: "My description",
				},
			},
		}, nil)
	assert.Error(t, err)
	assert.Contains(t, "release title is not set", err.Error())
	assert.Equal(t, sdk.StatusFail, res.Status)
}

func TestRunReleaseMissingReleaseNote(t *testing.T) {
	defer gock.Off()

	wk, ctx := setupTest(t)
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "cds.project",
			Value: "projKey",
		},
		{
			Name:  "cds.workflow",
			Value: "workflowName",
		},
		{
			Name:  "cds.run.number",
			Value: "999",
		},
	}...)
	res, err := RunRelease(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "artifacts",
					Value: "*.deb",
				},
				{
					Name:  "tag",
					Value: "1.1.1",
				},
				{
					Name:  "title",
					Value: "My title",
				},
			},
		}, nil)
	assert.Error(t, err)
	assert.Contains(t, "release note is not set", err.Error())
	assert.Equal(t, sdk.StatusFail, res.Status)
}

func TestRunReleaseMissingProjectKey(t *testing.T) {
	defer gock.Off()

	wk, ctx := setupTest(t)
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "cds.workflow",
			Value: "workflowName",
		},
		{
			Name:  "cds.run.number",
			Value: "999",
		},
	}...)
	res, err := RunRelease(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "artifacts",
					Value: "*.deb",
				},
				{
					Name:  "tag",
					Value: "1.1.1",
				},
				{
					Name:  "title",
					Value: "My title",
				},
				{
					Name:  "releaseNote",
					Value: "My description",
				},
			},
		}, nil)
	assert.Error(t, err)
	assert.Contains(t, "cds.project variable not found", err.Error())
	assert.Equal(t, sdk.StatusFail, res.Status)
}

func TestRunReleaseMissingWorkflowName(t *testing.T) {
	defer gock.Off()

	wk, ctx := setupTest(t)
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "cds.project",
			Value: "projKey",
		},
		{
			Name:  "cds.run.number",
			Value: "999",
		},
	}...)
	res, err := RunRelease(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "artifacts",
					Value: "*.deb",
				},
				{
					Name:  "tag",
					Value: "1.1.1",
				},
				{
					Name:  "title",
					Value: "My title",
				},
				{
					Name:  "releaseNote",
					Value: "My description",
				},
			},
		}, nil)
	assert.Error(t, err)
	assert.Contains(t, "cds.workflow variable not found", err.Error())
	assert.Equal(t, sdk.StatusFail, res.Status)
}

func TestRunReleaseMissingWorkflowRunNumber(t *testing.T) {
	defer gock.Off()

	wk, ctx := setupTest(t)
	wk.Params = append(wk.Params, []sdk.Parameter{
		{
			Name:  "cds.project",
			Value: "projKey",
		},
		{
			Name:  "cds.workflow",
			Value: "workflow Name",
		},
	}...)
	res, err := RunRelease(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "artifacts",
					Value: "*.deb",
				},
				{
					Name:  "tag",
					Value: "1.1.1",
				},
				{
					Name:  "title",
					Value: "My title",
				},
				{
					Name:  "releaseNote",
					Value: "My description",
				},
			},
		}, nil)
	assert.Error(t, err)
	assert.Contains(t, "cds.run.number variable not found", err.Error())
	assert.Equal(t, sdk.StatusFail, res.Status)
}
