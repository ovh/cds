package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ovh/cds/sdk"
)

func Test_getEnvRequirements(t *testing.T) {
	type args struct {
		requirements []sdk.Requirement
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "quote",
			args: args{requirements: []sdk.Requirement{{Name: "foo", Type: "Plugin", Value: "Bar with double \"quote"}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.EqualValues(t, "export FOO=\"Bar with double \\\"quote\"\n", getEnvRequirements(tt.args.requirements))
		})
	}
}
