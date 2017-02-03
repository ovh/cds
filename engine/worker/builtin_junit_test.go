package main

import (
	"reflect"
	"testing"

	"github.com/ovh/cds/sdk"
)

func Test_computeStats(t *testing.T) {
	type args struct {
		res *sdk.Result
		v   *sdk.Tests
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test with a malformed testcase",
			want: []string{
				"JUnit parser: testcase myTestCase has 3 failure(s)",
				"JUnit parser: testcase myTestCase has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 3 failure(s)",
				"JUnit parser: testsuite myTestSuite has 2 error(s)",
				"JUnit parser: testsuite myTestSuite has 1 test(s) failed",
			},
			args: args{
				res: &sdk.Result{},
				v: &sdk.Tests{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := computeStats(tt.args.res, tt.args.v); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("computeStats() = %v, want %v", got, tt.want)
			}
			if tt.args.res.Status != sdk.StatusFail {
				t.Errorf("status = %v, want %v", tt.args.res.Status, sdk.StatusFail)
			}
			if tt.args.v.TotalOK != 0 {
				t.Errorf("totalOK = %v, want %v", tt.args.v.TotalOK, 0)
			}
			// sum of failure + errors on testsuite attribute. So 1+1
			if tt.args.v.TotalKO != 2 {
				t.Errorf("totalKO = %v, want %v", tt.args.v.TotalKO, 2)
			}
			if tt.args.v.Total != 0 {
				t.Errorf("total = %v, want %v", tt.args.v.Total, 1)
			}
		})
	}
}
