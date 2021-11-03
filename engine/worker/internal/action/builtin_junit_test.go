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

	"github.com/ovh/venom"
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

	gock.New("http://lolcat.host").Post("/queue/workflows/666/test").
		Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
			switch mock.Request().URLStruct.String() {
			case "http://lolcat.host/queue/workflows/666/test":
				var report venom.Tests
				err := json.Unmarshal(bodyContent, &report)
				assert.NoError(t, err)
				assert.Equal(t, 3, report.Total)
				assert.Equal(t, 1, report.TotalKO)
				assert.Equal(t, 1, report.TotalSkipped)
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

	gock.New("http://lolcat.host").Post("/queue/workflows/666/test").
		Reply(200)

	var checkRequest gock.ObserverFunc = func(request *http.Request, mock gock.Mock) {
		bodyContent, err := io.ReadAll(request.Body)
		assert.NoError(t, err)
		request.Body = io.NopCloser(bytes.NewReader(bodyContent))
		if mock != nil {
			t.Logf("%s %s - Body: %s", mock.Request().Method, mock.Request().URLStruct.String(), string(bodyContent))
			switch mock.Request().URLStruct.String() {
			case "http://lolcat.host/queue/workflows/666/test":
				var report venom.Tests
				err := json.Unmarshal(bodyContent, &report)
				assert.NoError(t, err)
				assert.Equal(t, 3, report.Total)
				assert.Equal(t, 1, report.TotalKO)
				assert.Equal(t, 1, report.TotalSkipped)
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

func Test_ComputeStats(t *testing.T) {
	type args struct {
		res *sdk.Result
		v   *venom.Tests
	}
	tests := []struct {
		name                    string
		args                    args
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
			args: args{
				res: &sdk.Result{},
				v: &venom.Tests{
					TestSuites: []venom.TestSuite{
						{
							Name:     "myTestSuite",
							Errors:   0,
							Failures: 0,
							TestCases: []venom.TestCase{
								{
									Name: "myTestCase",
								},
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
			args: args{
				res: &sdk.Result{},
				v: &venom.Tests{
					TestSuites: []venom.TestSuite{
						{
							Name:     "myTestSuite",
							Errors:   0,
							Failures: 1,
							TestCases: []venom.TestCase{
								{
									Name:     "myTestCase",
									Failures: []venom.Failure{{Value: "Foo"}},
								},
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
			args: args{
				res: &sdk.Result{},
				v: &venom.Tests{
					TestSuites: []venom.TestSuite{
						{
							Errors:   0,
							Failures: 1,
							TestCases: []venom.TestCase{
								{
									Failures: []venom.Failure{{Value: "Foo"}},
								},
								{
									Failures: []venom.Failure{{Value: "Foo"}},
								},
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
			totalKO: 2, // sum of failure + errors on testsuite attribute. So 1+1
			total:   2,
			want: []string{
				"JUnit parser: 1 testsuite(s)",
				"JUnit parser: testsuite myTestSuite has 1 testcase(s)",
				"JUnit parser: testcase myTestCase has 3 failure(s)",
				"JUnit parser: testcase myTestCase has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 3 failure(s)",
				"JUnit parser: testsuite myTestSuite has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 1 test(s) failed",
			},
			args: args{
				res: &sdk.Result{},
				v: &venom.Tests{
					TestSuites: []venom.TestSuite{
						{
							Name:     "myTestSuite",
							Errors:   1,
							Failures: 1,
							TestCases: []venom.TestCase{
								{
									Name:     "myTestCase",
									Errors:   []venom.Failure{{Value: "Foo"}, {Value: "Foo"}},
									Failures: []venom.Failure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
								},
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
				"JUnit parser: testsuite myTestSuite has 6 failure(s)",
				"JUnit parser: testsuite myTestSuite has 4 error(s)",
				"JUnit parser: testsuite myTestSuite has 2 test(s) failed",
			},
			args: args{
				res: &sdk.Result{},
				v: &venom.Tests{
					TestSuites: []venom.TestSuite{
						{
							Name:     "myTestSuite",
							Errors:   1,
							Failures: 1,
							TestCases: []venom.TestCase{
								{
									Name:     "myTestCase 1",
									Errors:   []venom.Failure{{Value: "Foo"}, {Value: "Foo"}},
									Failures: []venom.Failure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
								},
								{
									Name:     "myTestCase 2",
									Errors:   []venom.Failure{{Value: "Foo"}, {Value: "Foo"}},
									Failures: []venom.Failure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComputeStats(tt.args.res, tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ComputeStats() = %v, want %v", got, tt.want)
			}
			if tt.args.res.Status != tt.status {
				t.Errorf("status = %v, want %v", tt.args.res.Status, tt.status)
			}

			if tt.args.v.TotalOK != tt.totalOK {
				t.Errorf("totalOK = %v, want %v", tt.args.v.TotalOK, tt.totalOK)
			}

			if tt.args.v.TotalKO != tt.totalKO {
				t.Errorf("totalKO = %v, want %v", tt.args.v.TotalKO, tt.totalKO)
			}
			if tt.args.v.Total != tt.total {
				t.Errorf("total = %v, want %v", tt.args.v.Total, tt.total)
			}
		})
	}
}
