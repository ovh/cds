package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func Test_processPipelineBuildJobParameter(t *testing.T) {
	type args struct {
		pbJob   *sdk.PipelineBuildJob
		secrets []sdk.Variable
	}
	testcases := []struct {
		name string
		args args
		want []sdk.Parameter
	}{
		{
			name: "Should replace .cds.app.xxx => .cds.env.yyy => password: zzz",
			args: args{
				pbJob: &sdk.PipelineBuildJob{
					Parameters: []sdk.Parameter{
						{
							Name:  "cds.app.xxx",
							Value: "{{.cds.env.yyy}}",
						},
					},
				},
				secrets: []sdk.Variable{
					{
						Name:  "cds.env.yyy",
						Value: "zzz",
					},
				},
			},
			want: []sdk.Parameter{
				{
					Name:  "cds.app.xxx",
					Value: "zzz",
				},
			},
		},
	}
	for _, tt := range testcases {
		processPipelineBuildJobParameter(tt.args.pbJob, tt.args.secrets)
		t.Log(tt.args.pbJob.Parameters)
		assert.EqualValues(t, tt.want, tt.args.pbJob.Parameters)
	}
}
