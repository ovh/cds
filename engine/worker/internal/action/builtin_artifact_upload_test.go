package action

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"
)

func TestRunArtifactUpload_Absolute(t *testing.T) {
	t.Cleanup(gock.Off)

	wk, ctx := SetupTest(t)
	wk.Params = []sdk.Parameter{
		{Name: "cds.project", Value: "project"},
	}

	assert.NoError(t, os.WriteFile("foo", []byte("something"), os.ModePerm))

	fi, err := os.Open("foo")
	require.NoError(t, err)
	fiPath, err := filepath.Abs(fi.Name())
	require.NoError(t, err)

	defer os.Remove("foo")

	gock.New("http://cds-cdn.local").Post("/item/upload").Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
		}
		assert.Equal(t, "http://cds-cdn.local/item/upload", mock.Request().URLStruct.String())
	}

	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPNoTimeoutClient())

	res, err := RunArtifactUpload(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Value: fiPath,
				}, {
					Name:  "tag",
					Value: "tag",
				},
			},
		},
		[]sdk.Variable{})

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
}

func TestRunArtifactUpload_Relative(t *testing.T) {
	t.Cleanup(gock.Off)

	wk, ctx := SetupTest(t)
	wk.Params = []sdk.Parameter{
		{Name: "cds.project", Value: "project"},
	}
	fname := filepath.Join(wk.workingDirectory.Name(), "foo")
	assert.NoError(t, afero.WriteFile(wk.workspace, fname, []byte("something"), os.ModePerm))

	gock.New("http://cds-cdn.local").Post("/item/upload").Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
		}
		assert.Equal(t, "http://cds-cdn.local/item/upload", mock.Request().URLStruct.String())
	}

	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPNoTimeoutClient())

	res, err := RunArtifactUpload(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Value: "foo",
				}, {
					Name:  "tag",
					Value: "tag",
				},
			},
		},
		[]sdk.Variable{})

	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
}
