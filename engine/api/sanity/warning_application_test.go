package sanity

import (
	"testing"

	"github.com/ovh/cds/engine/api/test"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func Test_checkApplicationVariable(t *testing.T) {
	log.SetLogger(t)

	type args struct {
		project  *sdk.Project
		app      *sdk.Application
		variable *sdk.Variable
	}
	tests := []struct {
		name    string
		args    args
		wantID  []int
		wantMsg []map[string]string
		wantErr bool
	}{
		{
			name: "All things empty",
			args: args{
				project:  &sdk.Project{},
				app:      &sdk.Application{},
				variable: &sdk.Variable{},
			},
		},
		{
			name: "Missing environment",
			args: args{
				project: &sdk.Project{},
				app: &sdk.Application{
					Name: "MyApp",
				},
				variable: &sdk.Variable{
					Value: "{{.cds.env.blabla}}",
				},
			},
			wantID: []int{MissingEnvironment},
			wantMsg: []map[string]string{
				{
					"ApplicationName": "MyApp",
				},
			},
			wantErr: false,
		},
		{
			name: "Missing environment 2",
			args: args{
				project: &sdk.Project{
					Environments: []sdk.Environment{
						{}, {},
					},
				},
				app: &sdk.Application{
					Name: "MyApp",
				},
				variable: &sdk.Variable{
					Value: "{{.cds.env.blabla}}",
				},
			},
			wantID: []int{MissingEnvironment, EnvironmentVariableUsedInApplicationDoesNotExist},
			wantMsg: []map[string]string{
				{
					"ApplicationName": "MyApp",
				},
				{
					"VarName":         "blabla",
					"ApplicationName": "MyApp",
				},
			},
			wantErr: false,
		},
		{
			name: "Bad vars",
			args: args{
				project: &sdk.Project{},
				app: &sdk.Application{
					Name: "MyApp",
				},
				variable: &sdk.Variable{
					Value: "{{cds.env.blabla}}",
				},
			},
			wantID: []int{InvalidVariableFormatUsedInApplication},
			wantMsg: []map[string]string{
				{
					"VarName":         "{{cds.env.blabla}}",
					"ApplicationName": "MyApp",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		got, err := checkApplicationVariable(tt.args.project, tt.args.app, tt.args.variable)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. checkApplicationVariable() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		gotID := []int{}
		for _, w := range got {
			gotID = append(gotID, int(w.ID))
		}
		test.EqualValuesWithoutOrder(t, tt.wantID, gotID, "%q. checkApplicationVariable() = %v, want %v", tt.name, gotID, tt.wantID)

		gotMsg := []map[string]string{}
		for _, w := range got {
			gotMsg = append(gotMsg, w.MessageParam)
		}
		test.EqualValuesWithoutOrder(t, tt.wantMsg, gotMsg, "%q. checkApplicationVariable() = %v, want %v", tt.name, gotMsg, tt.wantMsg)
	}
}
