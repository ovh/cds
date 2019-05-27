package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/sdk"
)

func Test_checkWorkflowPermissionsByUser(t *testing.T) {
	api, _, _, end := newTestAPI(t)
	defer end()

	type args struct {
		u     *sdk.User
		wName string
		pKey  string
		p     int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Should return true for user [read permission]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Permissions: sdk.UserPermissions{
						ProjectsPerm: map[string]int{
							"key1": 4,
						},
					},
				},
				wName: "workflow1",
				pKey:  "key1",
				p:     4,
			},
			want: true,
		},
		{
			name: "Should return false for user [read permission]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Permissions: sdk.UserPermissions{
						ProjectsPerm: map[string]int{
							"key1": 4,
						},
					},
				},
				wName: "workflow1",
				pKey:  "key2",
				p:     4,
			},
			want: false,
		},
		{
			name: "Should return true for user [write permission]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Permissions: sdk.UserPermissions{
						ProjectsPerm: map[string]int{
							"key1": 4,
						},
						WorkflowsPerm: sdk.UserPermissionsMap{
							sdk.UserPermissionKey("key1", "workflow1"): 7,
						},
					},
				},
				wName: "workflow1",
				pKey:  "key1",
				p:     7,
			},
			want: true,
		},
		{
			name: "Should return false for user [wrong project]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Permissions: sdk.UserPermissions{
						ProjectsPerm: map[string]int{
							"key1": 4,
						},
						WorkflowsPerm: sdk.UserPermissionsMap{
							sdk.UserPermissionKey("key2", "workflow1"): 7,
						},
					},
				},
				wName: "workflow1",
				pKey:  "key1",
				p:     7,
			},
			want: false,
		},
		{
			name: "Should return false for user [wrong workflow]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Permissions: sdk.UserPermissions{
						ProjectsPerm: map[string]int{
							"key1": 4,
						},
						WorkflowsPerm: sdk.UserPermissionsMap{
							sdk.UserPermissionKey("key2", "workflow1"): 7,
						},
					},
				},
				wName: "workflow1",
				pKey:  "key1",
				p:     7,
			},
			want: false,
		},
		{
			name: "Should return false for user [wrong permission]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Permissions: sdk.UserPermissions{
						ProjectsPerm: map[string]int{
							"key1": 4,
						},
						WorkflowsPerm: sdk.UserPermissionsMap{
							sdk.UserPermissionKey("key1", "workflow1"): 5,
						},
					},
				},
				wName: "workflow1",
				pKey:  "key1",
				p:     7,
			},
			want: false,
		},
		{
			name: "Should return true for user [execution]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Permissions: sdk.UserPermissions{
						ProjectsPerm: map[string]int{
							"key1": 4,
						},
						WorkflowsPerm: sdk.UserPermissionsMap{
							sdk.UserPermissionKey("key1", "workflow1"): 5,
						},
					},
				},
				wName: "workflow1",
				pKey:  "key1",
				p:     5,
			},
			want: true,
		},
		{
			name: "Should return false for user [execution]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Permissions: sdk.UserPermissions{
						ProjectsPerm: map[string]int{
							"key1": 4,
						},
						WorkflowsPerm: sdk.UserPermissionsMap{
							sdk.UserPermissionKey("key1", "workflow1"): 4,
						},
					},
				},
				wName: "workflow1",
				pKey:  "key1",
				p:     5,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		ctx := context.WithValue(context.Background(), auth.ContextUser, tt.args.u)
		m := map[string]string{}
		m["key"] = tt.args.pKey
		err := api.checkWorkflowPermissions(ctx, tt.args.wName, tt.args.p, m)
		got := err == nil
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
