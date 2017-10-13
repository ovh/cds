package api

import (
	"context"
	"reflect"
	"testing"

	"github.com/ovh/cds/engine/api/auth"
	"github.com/ovh/cds/sdk"
)

func Test_checkWorkerModelPermissionsByUser(t *testing.T) {
	api, _, _ := newTestAPI(t)

	type args struct {
		m *sdk.Model
		u *sdk.User
		p int
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Should return true for admin user",
			args: args{
				m: &sdk.Model{
					GroupID: 1,
				},
				u: &sdk.User{
					Admin: true,
				},
				p: 7,
			},
			want: true,
		},
		{
			name: "Should return true for user who has the right group for getting the model",
			args: args{
				m: &sdk.Model{
					GroupID: 1,
				},
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ID: 1,
						},
					},
				},
				p: 4,
			},
			want: true,
		},
		{
			name: "Should return false for user who has not the right group for updating the model",
			args: args{
				m: &sdk.Model{
					GroupID: 1,
				},
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ID: 1,
						},
					},
				},
				p: 7,
			},
			want: false,
		},
		{
			name: "Should return false for user who has not the right group",
			args: args{
				m: &sdk.Model{
					GroupID: 666,
				},
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ID: 1,
						},
					},
				},
				p: 7,
			},
			want: false,
		},
		{
			name: "Should return true for user who has the right group as admin for updating the model",
			args: args{
				m: &sdk.Model{
					GroupID: 1,
				},
				u: &sdk.User{
					ID:    1,
					Admin: false,
					Groups: []sdk.Group{
						{
							ID: 1,
							Admins: []sdk.User{
								{
									ID: 1,
								},
							},
						},
					},
				},
				p: 7,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		got := api.checkWorkerModelPermissionsByUser(tt.args.m, tt.args.u, tt.args.p)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_checkWorkflowPermissionsByUser(t *testing.T) {
	api, _, _ := newTestAPI(t)

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
					Groups: []sdk.Group{
						{
							ProjectGroups: []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
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
					Groups: []sdk.Group{
						{
							ProjectGroups: []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
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
					Groups: []sdk.Group{
						{
							ProjectGroups:  []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
							WorkflowGroups: []sdk.WorkflowGroup{{Permission: 7, Workflow: sdk.Workflow{ProjectKey: "key1", Name: "workflow1"}}},
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
					Groups: []sdk.Group{
						{
							ProjectGroups:  []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
							WorkflowGroups: []sdk.WorkflowGroup{{Permission: 7, Workflow: sdk.Workflow{ProjectKey: "key2", Name: "workflow1"}}},
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
					Groups: []sdk.Group{
						{
							ProjectGroups:  []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
							WorkflowGroups: []sdk.WorkflowGroup{{Permission: 7, Workflow: sdk.Workflow{ProjectKey: "key1", Name: "workflow2"}}},
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
					Groups: []sdk.Group{
						{
							ProjectGroups:  []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
							WorkflowGroups: []sdk.WorkflowGroup{{Permission: 5, Workflow: sdk.Workflow{ProjectKey: "key1", Name: "workflow1"}}},
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
					Groups: []sdk.Group{
						{
							ProjectGroups:  []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
							WorkflowGroups: []sdk.WorkflowGroup{{Permission: 5, Workflow: sdk.Workflow{ProjectKey: "key1", Name: "workflow1"}}},
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
					Groups: []sdk.Group{
						{
							ProjectGroups:  []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
							WorkflowGroups: []sdk.WorkflowGroup{{Permission: 4, Workflow: sdk.Workflow{ProjectKey: "key1", Name: "workflow1"}}},
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
		got := api.checkWorkflowPermissions(ctx, tt.args.wName, tt.args.p, m)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_checkApplicationPermissionsByUser(t *testing.T) {
	api, _, _ := newTestAPI(t)

	type args struct {
		u       *sdk.User
		appName string
		pKey    string
		p       int
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
					Groups: []sdk.Group{
						{
							ProjectGroups: []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
						},
					},
				},
				appName: "app1",
				pKey:    "key1",
				p:       4,
			},
			want: true,
		},
		{
			name: "Should return false for user [read permission]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ProjectGroups: []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
						},
					},
				},
				appName: "app1",
				pKey:    "key2",
				p:       4,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		ctx := context.WithValue(context.Background(), auth.ContextUser, tt.args.u)
		m := map[string]string{}
		m["key"] = tt.args.pKey
		got := api.checkApplicationPermissions(ctx, tt.args.appName, tt.args.p, m)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_checkPipelinePermissionsByUser(t *testing.T) {
	api, _, _ := newTestAPI(t)

	type args struct {
		u       *sdk.User
		pipName string
		pKey    string
		p       int
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
					Groups: []sdk.Group{
						{
							ProjectGroups: []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
						},
					},
				},
				pipName: "pip1",
				pKey:    "key1",
				p:       4,
			},
			want: true,
		},
		{
			name: "Should return false for user [read permission]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ProjectGroups: []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
						},
					},
				},
				pipName: "pip2",
				pKey:    "key2",
				p:       4,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		ctx := context.WithValue(context.Background(), auth.ContextUser, tt.args.u)
		m := map[string]string{}
		m["key"] = tt.args.pKey
		got := api.checkPipelinePermissions(ctx, tt.args.pipName, tt.args.p, m)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_checkEnvironmentPermissionsByUser(t *testing.T) {
	api, _, _ := newTestAPI(t)

	type args struct {
		u       *sdk.User
		envName string
		pKey    string
		p       int
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
					Groups: []sdk.Group{
						{
							ProjectGroups: []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
						},
					},
				},
				envName: "env1",
				pKey:    "key1",
				p:       4,
			},
			want: true,
		},
		{
			name: "Should return false for user [read permission]",
			args: args{
				u: &sdk.User{
					Admin: false,
					Groups: []sdk.Group{
						{
							ProjectGroups: []sdk.ProjectGroup{{Permission: 4, Project: sdk.Project{Key: "key1"}}},
						},
					},
				},
				envName: "env2",
				pKey:    "key2",
				p:       4,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		ctx := context.WithValue(context.Background(), auth.ContextUser, tt.args.u)
		m := map[string]string{}
		m["key"] = tt.args.pKey
		got := api.checkPipelinePermissions(ctx, tt.args.envName, tt.args.p, m)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
