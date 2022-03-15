package action

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/h2non/gock.v1"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
)

func TestRunParseJunitTestResultAction_Absolute(t *testing.T) {
	fileContent := `<?xml version="1.0" encoding="UTF-8"?>
	<testsuites>
	   <testsuite name="JUnitXmlReporter" errors="0" tests="0" failures="0" time="0" timestamp="2013-05-24T10:23:58" />
	   <testsuite name="JUnitXmlReporter.constructor" errors="0" skipped="1" tests="3" failures="1" time="0.006" timestamp="2013-05-24T10:23:58">
		  <properties>
			 <property name="java.vendor" value="Sun Microsystems Inc." />
			 <property name="compiler.debug" value="on" />
			 <property name="project.jdk.classpath" value="jdk.classpath.1.6" />
		  </properties>
		  <testcase classname="JUnitXmlReporter.constructor" name="should default path to an empty string" time="0.006">
			 <failure message="test failure">Assertion failed</failure>
		  </testcase>
		  <testcase classname="JUnitXmlReporter.constructor" name="should default consolidate to true" time="0">
			 <skipped />
		  </testcase>
		  <testcase classname="JUnitXmlReporter.constructor" name="should default useDotNotation to true" time="0" />
	   </testsuite>
	</testsuites>`

	defer gock.Off()

	wk, ctx := SetupTest(t)
	assert.NoError(t, os.WriteFile("results.xml", []byte(fileContent), os.ModePerm))
	defer os.RemoveAll("results.xml")

	fi, err := os.Open("results.xml")
	require.NoError(t, err)
	fiPath, err := filepath.Abs(fi.Name())
	require.NoError(t, err)

	gock.New("http://cds-api.local").Post("/queue/workflows/666/test").
		Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
			switch mock.Request().URLStruct.String() {
			case "http://cds-api.local/queue/workflows/666/test":
				var report sdk.JUnitTestsSuites
				require.NoError(t, json.Unmarshal(bodyContent, &report))
				report = report.EnsureData()
				stats := report.ComputeStats()
				assert.Equal(t, 3, stats.Total)
				assert.Equal(t, 1, stats.TotalKO)
				assert.Equal(t, 1, stats.TotalSkipped)
			}
		}
	}

	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPNoTimeoutClient())
	res, err := RunParseJunitTestResultAction(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Value: fiPath,
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusFail, res.Status)
}

func TestRunParseJunitTestResultAction_Relative(t *testing.T) {
	fileContent := `<?xml version="1.0" encoding="UTF-8"?>
	<testsuites>
	   <testsuite name="JUnitXmlReporter" errors="0" tests="0" failures="0" time="0" timestamp="2013-05-24T10:23:58" />
	   <testsuite name="JUnitXmlReporter.constructor" errors="0" skipped="1" tests="3" failures="1" time="0.006" timestamp="2013-05-24T10:23:58">
		  <properties>
			 <property name="java.vendor" value="Sun Microsystems Inc." />
			 <property name="compiler.debug" value="on" />
			 <property name="project.jdk.classpath" value="jdk.classpath.1.6" />
		  </properties>
		  <testcase classname="JUnitXmlReporter.constructor" name="should default path to an empty string" time="0.006">
			 <failure message="test failure">Assertion failed</failure>
		  </testcase>
		  <testcase classname="JUnitXmlReporter.constructor" name="should default consolidate to true" time="0">
			 <skipped />
		  </testcase>
		  <testcase classname="JUnitXmlReporter.constructor" name="should default useDotNotation to true" time="0" />
	   </testsuite>
	</testsuites>`

	defer gock.Off()

	wk, ctx := SetupTest(t)
	fname := filepath.Join(wk.workingDirectory.Name(), "results.xml")
	require.NoError(t, afero.WriteFile(wk.BaseDir(), fname, []byte(fileContent), os.ModePerm))

	gock.New("http://cds-api.local").Post("/queue/workflows/666/test").
		Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
			switch mock.Request().URLStruct.String() {
			case "http://cds-api.local/queue/workflows/666/test":
				var report sdk.JUnitTestsSuites
				require.NoError(t, json.Unmarshal(bodyContent, &report))
				report = report.EnsureData()
				stats := report.ComputeStats()
				assert.Equal(t, 3, stats.Total)
				assert.Equal(t, 1, stats.TotalKO)
				assert.Equal(t, 1, stats.TotalSkipped)
			}
		}
	}

	gock.Observe(checkRequest)

	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPClient())
	gock.InterceptClient(wk.Client().(cdsclient.Raw).HTTPNoTimeoutClient())
	res, err := RunParseJunitTestResultAction(ctx, wk,
		sdk.Action{
			Parameters: []sdk.Parameter{
				{
					Name:  "path",
					Value: "results.xml",
				},
			},
		}, nil)
	assert.NoError(t, err)
	assert.Equal(t, sdk.StatusFail, res.Status)
}

func Test_EnsureAndComputeStats(t *testing.T) {
	tests := []struct {
		name                    string
		args                    sdk.JUnitTestsSuites
		want                    []string
		status                  string
		totalOK, totalKO, total int
	}{
		{
			name:    "success",
			status:  sdk.StatusSuccess,
			totalOK: 1,
			totalKO: 0,
			total:   1,
			want: []string{
				"JUnit parser: 1 testsuite(s)",
				"JUnit parser: testsuite myTestSuite has 1 testcase(s)",
			},
			args: sdk.JUnitTestsSuites{
				TestSuites: []sdk.JUnitTestSuite{
					{
						Name:     "myTestSuite",
						Errors:   0,
						Failures: 0,
						TestCases: []sdk.JUnitTestCase{
							{
								Name: "myTestCase",
							},
						},
					},
				},
			},
		},
		{
			name:    "failed",
			status:  sdk.StatusFail,
			totalOK: 0,
			totalKO: 1, // sum of failure + errors on testsuite attribute. So 1+1
			total:   1,
			want: []string{
				"JUnit parser: 1 testsuite(s)",
				"JUnit parser: testsuite myTestSuite has 1 testcase(s)",
				"JUnit parser: testcase myTestCase has 1 failure(s)",
				"JUnit parser: testsuite myTestSuite has 1 failure(s)",
				"JUnit parser: testsuite myTestSuite has 1 test(s) failed",
			},
			args: sdk.JUnitTestsSuites{
				TestSuites: []sdk.JUnitTestSuite{
					{
						Name:     "myTestSuite",
						Errors:   0,
						Failures: 1,
						TestCases: []sdk.JUnitTestCase{
							{
								Name:     "myTestCase",
								Failures: []sdk.JUnitTestFailure{{Value: "Foo"}},
							},
						},
					},
				},
			},
		},
		{
			name:    "defaultName",
			status:  sdk.StatusFail,
			totalOK: 0,
			totalKO: 2,
			total:   2,
			want: []string{
				"JUnit parser: 1 testsuite(s)",
				"JUnit parser: testsuite TestSuite.0 has 2 testcase(s)",
				"JUnit parser: testcase TestCase.0 has 1 failure(s)",
				"JUnit parser: testcase TestCase.1 has 1 failure(s)",
				"JUnit parser: testsuite TestSuite.0 has 2 failure(s)",
				"JUnit parser: testsuite TestSuite.0 has 2 test(s) failed",
			},
			args: sdk.JUnitTestsSuites{
				TestSuites: []sdk.JUnitTestSuite{
					{
						Errors:   0,
						Failures: 1,
						TestCases: []sdk.JUnitTestCase{
							{
								Failures: []sdk.JUnitTestFailure{{Value: "Foo"}},
							},
							{
								Failures: []sdk.JUnitTestFailure{{Value: "Foo"}},
							},
						},
					},
				},
			},
		},
		{
			name:    "malformed",
			status:  sdk.StatusFail,
			totalOK: 0,
			totalKO: 1,
			total:   1,
			want: []string{
				"JUnit parser: 1 testsuite(s)",
				"JUnit parser: testsuite myTestSuite has 1 testcase(s)",
				"JUnit parser: testcase myTestCase has 3 failure(s)",
				"JUnit parser: testcase myTestCase has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 1 error(s)",
				"JUnit parser: testsuite myTestSuite has 1 test(s) failed",
			},
			args: sdk.JUnitTestsSuites{
				TestSuites: []sdk.JUnitTestSuite{
					{
						Name:     "myTestSuite",
						Errors:   1,
						Failures: 1,
						TestCases: []sdk.JUnitTestCase{
							{
								Name:     "myTestCase",
								Errors:   []sdk.JUnitTestFailure{{Value: "Foo"}, {Value: "Foo"}},
								Failures: []sdk.JUnitTestFailure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
							},
						},
					},
				},
			},
		},
		{
			name:    "malformedBis",
			status:  sdk.StatusFail,
			totalOK: 0,
			totalKO: 2,
			total:   2,
			want: []string{
				"JUnit parser: 1 testsuite(s)",
				"JUnit parser: testsuite myTestSuite has 2 testcase(s)",
				"JUnit parser: testcase myTestCase 1 has 3 failure(s)",
				"JUnit parser: testcase myTestCase 1 has 2 error(s)",
				"JUnit parser: testcase myTestCase 2 has 3 failure(s)",
				"JUnit parser: testcase myTestCase 2 has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 2 test(s) failed",
			},
			args: sdk.JUnitTestsSuites{
				TestSuites: []sdk.JUnitTestSuite{
					{
						Name:     "myTestSuite",
						Errors:   1,
						Failures: 1,
						TestCases: []sdk.JUnitTestCase{
							{
								Name:     "myTestCase 1",
								Errors:   []sdk.JUnitTestFailure{{Value: "Foo"}, {Value: "Foo"}},
								Failures: []sdk.JUnitTestFailure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
							},
							{
								Name:     "myTestCase 2",
								Errors:   []sdk.JUnitTestFailure{{Value: "Foo"}, {Value: "Foo"}},
								Failures: []sdk.JUnitTestFailure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tests := tt.args.EnsureData()
			stats := tests.ComputeStats()
			reasons := ComputeTestsReasons(tests)
			if !reflect.DeepEqual(reasons, tt.want) {
				t.Fatalf("ComputeStats() = %v, want %v", reasons, tt.want)
			}
			require.Equal(t, tt.totalOK, stats.TotalOK)
			require.Equal(t, tt.totalKO, stats.TotalKO)
			require.Equal(t, tt.total, stats.Total)
		})
	}
}
