package main

import (
	"reflect"
	"testing"

	"github.com/ovh/cds/sdk"
)

func Test_checkWorkerModelPermissionsByUser(t *testing.T) {
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
		got := checkWorkerModelPermissionsByUser(tt.args.m, tt.args.u, tt.args.p)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. checkWorkerModelPermissionsByUser() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
