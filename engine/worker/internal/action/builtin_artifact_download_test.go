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

	f1 := bytes.NewBufferString("contentfile")

	gock.New("http://cds-cdn.local").Get("/item/run-result/the-ref/download").
		Reply(200).Body(f1)

	ref := sdk.CDNRunResultAPIRef{
		ProjectKey:    "projKey",
		WorkflowName:  "WorkflowName",
		ArtifactName:  "myFile.txt",
		RunResultType: sdk.WorkflowRunResultTypeArtifact,
	}

	it := sdk.CDNItemLinks{
		CDNHttpURL: "http://lolcat.cdn.host",
		Items: []sdk.CDNItem{
			{
				ID:         "1",
				Type:       sdk.CDNTypeItemRunResult,
				APIRef:     &ref,
				APIRefHash: "the-ref",
				MD5:        "d62dd48969b2bcf4023f51be7cc02c05",
			},
		},
	}

	gock.New("http://cds-api.local").Get("/project/projKey/workflows/workflowName/runs/999/artifacts/links").
		Reply(200).JSON(it)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPNoTimeoutClient())

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
		{
			Name:  "cds.build.pkg",
			Value: "myFile.txt",
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
					Value: "{{.cds.build.pkg}}",
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
	f1 := bytes.NewBufferString("contentfile")

	gock.New("http://cds-cdn.local").Get("/item/run-result/the-ref/download").
		Reply(200).Body(f1)

	ref := sdk.CDNRunResultAPIRef{
		ProjectKey:    "projKey",
		WorkflowName:  "WorkflowName",
		ArtifactName:  fileName,
		RunResultType: sdk.WorkflowRunResultTypeArtifact,
	}

	it := sdk.CDNItemLinks{
		CDNHttpURL: "http://lolcat.cdn.host",
		Items: []sdk.CDNItem{
			{
				ID:         "1",
				Type:       sdk.CDNTypeItemRunResult,
				APIRef:     &ref,
				APIRefHash: "the-ref",
				MD5:        "d62dd48969b2bcf4023f51be7cc02c05",
			},
		},
	}

	gock.New("http://cds-api.local").Get("/project/projKey/workflows/workflowName/runs/999/artifacts/links").
		Reply(200).JSON(it)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPNoTimeoutClient())

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
