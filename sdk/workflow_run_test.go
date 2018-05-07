package sdk

import (
	"reflect"
	"testing"
)

func TestArtifactsGetUniqueNameAndLatest(t *testing.T) {
	type args struct {
		in []WorkflowNodeRunArtifact
	}
	tests := []struct {
		name string
		args args
		want []WorkflowNodeRunArtifact
	}{
		{
			name: "simple",
			args: args{[]WorkflowNodeRunArtifact{{ID: 1, Name: "foo"}, {ID: 2, Name: "foo"}, {ID: 3, Name: "bar"}}},
			want: []WorkflowNodeRunArtifact{{ID: 2, Name: "foo"}, {ID: 3, Name: "bar"}},
		},
		{
			name: "simple2",
			args: args{[]WorkflowNodeRunArtifact{{ID: 1, Name: "foo"}, {ID: 2, Name: "foo2"}, {ID: 3, Name: "bar"}}},
			want: []WorkflowNodeRunArtifact{{ID: 1, Name: "foo"}, {ID: 2, Name: "foo2"}, {ID: 3, Name: "bar"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ArtifactsGetUniqueNameAndLatest(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ArtifactsGetUniqueNameAndLatest() %s = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
