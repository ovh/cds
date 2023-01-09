package workflow

import (
	"context"
	"github.com/stretchr/testify/require"
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

func TestGetWorkerModelv2Error(t *testing.T) {
	wr := sdk.WorkflowRun{

		Workflow: sdk.Workflow{
			WorkflowData: sdk.WorkflowData{
				Node: sdk.Node{
					Context: &sdk.NodeContext{
						ApplicationID: 2,
					},
				},
			},
			Applications: map[int64]sdk.Application{
				1: {
					ID:                 1,
					VCSServer:          "",
					RepositoryFullname: "ovh/cds",
				},
			},
		},
	}

	_, _, err := processNodeJobRunRequirementsGetModelV2(context.TODO(), nil, "PROJ", wr, "cds/mymodel")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to find repository for this worker model")

	_, _, err = processNodeJobRunRequirementsGetModelV2(context.TODO(), nil, "PROJ", wr, "rien/proj/vcs/ovh/cds/mymodel")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to handle the worker model requirement")

	_, _, err = processNodeJobRunRequirementsGetModelV2(context.TODO(), nil, "PROJ", wr, "mymodel")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to retrieve worker model data because the workflow root pipeline does not contain application in context")

	wr.Workflow.WorkflowData.Node.Context.ApplicationID = 1
	_, _, err = processNodeJobRunRequirementsGetModelV2(context.TODO(), nil, "PROJ", wr, "mymodel")
	require.Error(t, err)
	t.Logf("%+v", err)
	require.Contains(t, err.Error(), "unable to retrieve worker model data because the workflow root pipeline does not contain any vcs configuration")

}
