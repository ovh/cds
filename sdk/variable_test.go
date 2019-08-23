package sdk_test

import (
	"reflect"
	"testing"

	"github.com/ovh/cds/sdk"
)

func Test_EnvVartoENV(t *testing.T) {
	tests := []struct {
		name string
		args sdk.Parameter
		want []string
	}{
		{
			args: sdk.Parameter{
				Name:  "cds.env.MyStringVariable",
				Value: "value",
			},
			want: []string{
				"CDS_ENV_MYSTRINGVARIABLE=value",
				"CDS_ENV_MyStringVariable=value",
				"MyStringVariable=value",
				"MYSTRINGVARIABLE=value",
			},
		},
		{
			args: sdk.Parameter{
				Name:  "cds.env.My.String.Variable",
				Value: "value",
			},
			want: []string{
				"CDS_ENV_MY_STRING_VARIABLE=value",
				"CDS_ENV_My.String.Variable=value",
				"My.String.Variable=value",
				"MY_STRING_VARIABLE=value",
			},
		},
		{
			args: sdk.Parameter{
				Name:  "cds.env.My-String-Variable",
				Value: "value",
			},
			want: []string{
				"CDS_ENV_MY_STRING_VARIABLE=value",
				"CDS_ENV_My-String-Variable=value",
				"My-String-Variable=value",
				"MY_STRING_VARIABLE=value",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sdk.EnvVartoENV(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sdk.EnvVartoENV() = %v, want %v", got, tt.want)
			}
		})
	}
}
