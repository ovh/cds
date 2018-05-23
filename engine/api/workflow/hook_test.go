package workflow

import (
	"reflect"
	"testing"

	"github.com/ovh/cds/sdk"
)

func Test_mergeAndDiffHook(t *testing.T) {
	type args struct {
		oldHooks map[string]sdk.WorkflowNodeHook
		newHooks map[string]sdk.WorkflowNodeHook
	}
	tests := []struct {
		name             string
		args             args
		wantHookToUpdate map[string]sdk.WorkflowNodeHook
		wantHookToDelete map[string]sdk.WorkflowNodeHook
	}{
		{
			name: "one to update",
			args: args{
				oldHooks: map[string]sdk.WorkflowNodeHook{
					"my-uuid-a": sdk.WorkflowNodeHook{Ref: "AAA", UUID: "my-uuid-a"},
				},
				newHooks: map[string]sdk.WorkflowNodeHook{
					"my-uuid-b": sdk.WorkflowNodeHook{Ref: "BBB"},
					"my-uuid-a": sdk.WorkflowNodeHook{Ref: "AAA", UUID: "my-uuid-a"},
				},
			},
			wantHookToUpdate: map[string]sdk.WorkflowNodeHook{
				"my-uuid-b": sdk.WorkflowNodeHook{Ref: "BBB"},
			},
			wantHookToDelete: map[string]sdk.WorkflowNodeHook{},
		},
		{
			name: "one delete",
			args: args{
				oldHooks: map[string]sdk.WorkflowNodeHook{
					"my-uuid-a": sdk.WorkflowNodeHook{Ref: "AAA", UUID: "my-uuid-a"},
				},
				newHooks: map[string]sdk.WorkflowNodeHook{},
			},
			wantHookToUpdate: map[string]sdk.WorkflowNodeHook{},
			wantHookToDelete: map[string]sdk.WorkflowNodeHook{
				"my-uuid-a": sdk.WorkflowNodeHook{Ref: "AAA", UUID: "my-uuid-a"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHookToUpdate, gotHookToDelete := mergeAndDiffHook(tt.args.oldHooks, tt.args.newHooks)
			if !reflect.DeepEqual(gotHookToUpdate, tt.wantHookToUpdate) {
				t.Errorf("mergeAndDiffHook() gotHookToUpdate = %v, want %v", gotHookToUpdate, tt.wantHookToUpdate)
			}
			if !reflect.DeepEqual(gotHookToDelete, tt.wantHookToDelete) {
				t.Errorf("mergeAndDiffHook() gotHookToDelete = %v, want %v", gotHookToDelete, tt.wantHookToDelete)
			}
		})
	}
}
