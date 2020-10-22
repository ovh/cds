package action

import (
	"bytes"
	"io/ioutil"
	"mime"
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

	assert.NoError(t, ioutil.WriteFile("foo", []byte("something"), os.ModePerm))

	fi, err := os.Open("foo")
	require.NoError(t, err)
	fiPath, err := filepath.Abs(fi.Name())
	require.NoError(t, err)

	defer os.Remove("foo")

	gock.New("http://lolcat.host").Get("/project/project/storage/shared.infra").
		Reply(200)

	gock.New("http://lolcat.host").Post("/project/project/storage/shared.infra/artifact/dGFn").
		Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := ioutil.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = ioutil.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
		}
		switch mock.Request().URLStruct.String() {
		case "http://lolcat.host/queue/workflows/666/coverage":
			require.NoError(t, request.ParseMultipartForm(10000))

			_, params, err := mime.ParseMediaType(request.Header.Get("Content-Disposition"))
			assert.NoError(t, err)

			fileName := params["filename"]
			assert.Equal(t, "foo", fileName)

			md5 := params["md4sum"]
			assert.Equal(t, "32c0c0a755c70c6faef2eeb98b66c3aeee4c389d62bb9b639796c37abe3d", md5)

		case "/project/project/storage/shared.infra/artifact":
		}
	}

	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPSSEClient())

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

	gock.New("http://lolcat.host").Get("/project/project/storage/shared.infra").
		Reply(200)

	gock.New("http://lolcat.host").Post("/project/project/storage/shared.infra/artifact/dGFn").
		Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := ioutil.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = ioutil.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
		}
		switch mock.Request().URLStruct.String() {
		case "http://lolcat.host/queue/workflows/666/coverage":
			require.NoError(t, request.ParseMultipartForm(10000))

			_, params, err := mime.ParseMediaType(request.Header.Get("Content-Disposition"))
			assert.NoError(t, err)

			fileName := params["filename"]
			assert.Equal(t, "foo", fileName)

			md5 := params["md4sum"]
			assert.Equal(t, "32c0c0a755c70c6faef2eeb98b66c3aeee4c389d62bb9b639796c37abe3d", md5)

		case "/project/project/storage/shared.infra/artifact":
		}
	}

	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPSSEClient())

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
