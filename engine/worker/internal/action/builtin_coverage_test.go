package action

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/sguiheux/go-coverage"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/sdk"
	"github.com/stretchr/testify/assert"
)

func TestRunCoverage_Absolute(t *testing.T) {
	defer gock.Off()

	wk, ctx := SetupTest(t)
	assert.NoError(t, os.WriteFile("results.xml", []byte(cobertura_result), os.ModePerm))
	defer os.RemoveAll("results.xml")

	fi, err := os.Open("results.xml")
	require.NoError(t, err)
	fiPath, err := filepath.Abs(fi.Name())
	require.NoError(t, err)

	gock.New("http://cds-cdn.local").Post("/item/upload").Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
			switch mock.Request().URLStruct.String() {
			case "http://cds-api.local/queue/workflows/666/coverage":
				var report coverage.Report
				err := json.Unmarshal(bodyContent, &report)
				assert.NoError(t, err)
				require.Equal(t, 8, report.TotalLines)
				require.Equal(t, 6, report.CoveredLines)
				require.Equal(t, 4, report.TotalBranches)
				require.Equal(t, 2, report.CoveredBranches)
			}
		}
	}

	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPNoTimeoutClient())
	res, err := RunParseCoverageResultAction(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Value: fiPath,
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
}

func TestRunCoverage_Relative(t *testing.T) {
	defer gock.Off()

	wk, ctx := SetupTest(t)
	fname := filepath.Join(wk.workingDirectory.Name(), "results.xml")
	require.NoError(t, afero.WriteFile(wk.BaseDir(), fname, []byte(cobertura_result), os.ModePerm))

	gock.New("http://cds-cdn.local").Post("/item/upload").Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
		}
	}

	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPNoTimeoutClient())
	res, err := RunParseCoverageResultAction(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Value: "results.xml",
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusSuccess, res.Status)
}

const cobertura_result = `<?xml version="1.0" ?>
<!DOCTYPE coverage SYSTEM "http://cobertura.sourceforge.net/xml/coverage-04.dtd">
<coverage lines-valid="8"  lines-covered="6"  line-rate="1"  branches-valid="4"  branches-covered="2"  branch-rate="1"  timestamp="1394890504210" complexity="0" version="0.1">
    <sources>
        <source>/Users/leobalter/dev/testing/solutions/3</source>
    </sources>
    <packages>
        <package name="3"  line-rate="1"  branch-rate="1" >
            <classes>
                <class name="cc.js"  filename="cc.js"  line-rate="1"  branch-rate="1" >
                    <methods>
                        <method name="normalize"  hits="11"  signature="()V" >
                            <lines><line number="1"  hits="11" /></lines>
                        </method>
                        <method name="getBrand"  hits="7"  signature="()V" >
                            <lines><line number="5"  hits="7" /></lines>
                        </method>
                    </methods>
                    <lines>
                        <line number="1"  hits="1"  branch="false" />
                        <line number="2"  hits="11"  branch="false" />
                        <line number="5"  hits="1"  branch="false" />
                        <line number="6"  hits="7"  branch="false" />
                        <line number="15"  hits="7"  branch="false" />
                        <line number="17"  hits="7"  branch="false" />
                        <line number="18"  hits="25"  branch="true"  condition-coverage="100% (4/4)" />
                        <line number="20"  hits="6"  branch="false" />
                    </lines>
                </class>
            </classes>
        </package>
    </packages>
</coverage>

`
