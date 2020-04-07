package action

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func TestRunArtifactDownload(t *testing.T) {
	defer gock.Off()

	wk, ctx := SetupTest(t)

	as := []sdk.WorkflowNodeRunArtifact{
		sdk.WorkflowNodeRunArtifact{
			ID:   1,
			Name: "myFile.txt",
			Tag:  "999",
		},
		sdk.WorkflowNodeRunArtifact{
			ID:   2,
			Name: "myFile.csv",
			Tag:  "999",
		},
	}

	f1 := bytes.NewBufferString("contentfile")

	gock.New("http://lolcat.host").Get("/project/projKey/workflows/workflowName/runs/999/artifacts").
		Reply(200).JSON(as)
	gock.New("http://lolcat.host").Get("/project/projKey/workflows/workflowName/artifact/1").
		Reply(200).Body(f1)

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
		{
			Name:  "cds.version",
			Value: "1",
		},
	}...)
	res, err := RunArtifactDownload(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Value: "./tmp/cds-tests",
				},
				{
					Name:  "pattern",
					Value: ".*.txt",
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)

	assert.FileExists(t, filepath.Join(wk.workingDirectory.File.Name(), "./tmp/cds-tests/myFile.txt"))

	_, err = os.Lstat(filepath.Join(wk.workingDirectory.File.Name(), "./tmp/cds-tests/myFile.csv"))
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))
}

func TestRunArtifactDownloadOutsideWorkspace(t *testing.T) {
	defer gock.Off()

	wk, ctx := SetupTest(t)

	fileName := sdk.RandomString(10)

	as := []sdk.WorkflowNodeRunArtifact{
		sdk.WorkflowNodeRunArtifact{
			ID:   1,
			Name: fileName,
			Tag:  "999",
		},
	}

	f1 := bytes.NewBufferString("contentfile")

	gock.New("http://lolcat.host").Get("/project/projKey/workflows/workflowName/runs/999/artifacts").
		Reply(200).JSON(as)
	gock.New("http://lolcat.host").Get("/project/projKey/workflows/workflowName/artifact/1").
		Reply(200).Body(f1)

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
		{
			Name:  "cds.version",
			Value: "1",
		},
	}...)
	res, err := RunArtifactDownload(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Value: "/tmp/cds-tests",
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)

	assert.FileExists(t, "/tmp/cds-tests/"+fileName)
}
