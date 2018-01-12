package workflow

import (
	"reflect"
	"testing"

	"github.com/ovh/cds/sdk"
)

func Test_prepareRequirementsToNodeJobRunParameters(t *testing.T) {
	type args struct {
		reqs sdk.RequirementList
	}
	tests := []struct {
		name string
		args args
		want []sdk.Parameter
	}{
		{
			name: "test add reqs to params",
			args: args{reqs: sdk.RequirementList{{Name: "git", Type: sdk.BinaryRequirement, Value: "git"}}},
			want: []sdk.Parameter{{Name: "job.requirement.binary.git", Type: "string", Value: "git"}},
		},
		{
			name: "test add reqs to params with service",
			args: args{reqs: sdk.RequirementList{{Name: "mypg", Type: sdk.ServiceRequirement, Value: "postgres:9.2 user=aa password=bb"}}},
			want: []sdk.Parameter{
				{Name: "job.requirement.service.mypg.image", Type: "string", Value: "postgres:9.2"},
				{Name: "job.requirement.service.mypg.options", Type: "string", Value: "user=aa password=bb"},
				{Name: "job.requirement.service.mypg", Type: "string", Value: "postgres:9.2 user=aa password=bb"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := prepareRequirementsToNodeJobRunParameters(tt.args.reqs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("prepareRequirementsToNodeJobRunParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}
