package main

import (
	"reflect"
	"testing"

	"github.com/ovh/cds/engine/venom"
	"github.com/ovh/cds/sdk"
)

func Test_computeStats(t *testing.T) {

	t.SkipNow()
	type args struct {
		res *sdk.Result
		v   *venom.Tests
	}
	tests := []struct {
		name                    string
		args                    args
		want                    []string
		status                  sdk.Status
		totalOK, totalKO, total int
	}{
		{
			name:    "success",
			status:  sdk.StatusSuccess,
			totalOK: 1,
			totalKO: 0,
			total:   1,
			want:    []string{},
			args: args{
				res: &sdk.Result{},
				v: &venom.Tests{
					TestSuites: []sdk.TestSuite{
						{
							Name:     "myTestSuite",
							Errors:   0,
							Failures: 0,
							TestCases: []sdk.TestCase{
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
				"JUnit parser: testcase myTestCase has 1 failure(s)",
				"JUnit parser: testsuite myTestSuite has 1 failure(s)",
				"JUnit parser: testsuite myTestSuite has 1 test(s) failed",
			},
			args: args{
				res: &sdk.Result{},
				v: &venom.Tests{
					TestSuites: []sdk.TestSuite{
						{
							Name:     "myTestSuite",
							Errors:   0,
							Failures: 1,
							TestCases: []sdk.TestCase{
								{
									Name:     "myTestCase",
									Failures: []sdk.Failure{{Value: "Foo"}},
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
				"JUnit parser: testcase myTestCase has 3 failure(s)",
				"JUnit parser: testcase myTestCase has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 3 failure(s)",
				"JUnit parser: testsuite myTestSuite has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 1 test(s) failed",
			},
			args: args{
				res: &sdk.Result{},
				v: &venom.Tests{
					TestSuites: []sdk.TestSuite{
						{
							Name:     "myTestSuite",
							Errors:   1,
							Failures: 1,
							TestCases: []sdk.TestCase{
								{
									Name:     "myTestCase",
									Errors:   []sdk.Failure{{Value: "Foo"}, {Value: "Foo"}},
									Failures: []sdk.Failure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
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
					TestSuites: []sdk.TestSuite{
						{
							Name:     "myTestSuite",
							Errors:   1,
							Failures: 1,
							TestCases: []sdk.TestCase{
								{
									Name:     "myTestCase 1",
									Errors:   []sdk.Failure{{Value: "Foo"}, {Value: "Foo"}},
									Failures: []sdk.Failure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
								},
								{
									Name:     "myTestCase 2",
									Errors:   []sdk.Failure{{Value: "Foo"}, {Value: "Foo"}},
									Failures: []sdk.Failure{{Value: "Foo"}, {Value: "Foo"}, {Value: "Foo"}},
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
			if got := computeStats(tt.args.res, tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("computeStats() = %v, want %v", got, tt.want)
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
