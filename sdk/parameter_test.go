package sdk

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParameterMapMerge(t *testing.T) {
	original := map[string]string{
		"test":     "value1",
		"default":  "ok",
		"git.hash": "XXX",
	}
	override := map[string]string{
		"test":     "override",
		"git.hash": "XXXBIS",
	}

	res := ParametersMapMerge(original, override)
	assert.Equal(t, "override", res["test"])
	assert.Equal(t, "ok", res["default"])
	assert.Equal(t, "XXXBIS", res["git.hash"])
}

func TestParameterMapMerge_WithExcludeGitParams(t *testing.T) {
	original := map[string]string{
		"test":     "value1",
		"default":  "ok",
		"git.hash": "XXX",
	}
	override := map[string]string{
		"test":     "override",
		"git.hash": "XXXBIS",
	}

	res := ParametersMapMerge(original, override, MapMergeOptions.ExcludeGitParams)
	assert.Equal(t, "override", res["test"])
	assert.Equal(t, "ok", res["default"])
	assert.Equal(t, "XXX", res["git.hash"])
}

func TestParametersMerge(t *testing.T) {
	original := []Parameter{
		Parameter{Name: "test", Value: "value1"},
		Parameter{Name: "default", Value: "ok"},
	}
	override := []Parameter{
		Parameter{Name: "test", Value: "override"},
	}

	res := ParametersMerge(original, override)
	for _, param := range res {
		if param.Name == "test" {
			assert.Equal(t, "override", param.Value)
			continue
		}
		if param.Name == "default" {
			assert.Equal(t, "ok", param.Value)
		}
	}
}

func TestParametersToMapWithPrefixBuiltinVar(t *testing.T) {
	type args struct {
		prefix string
		params []Parameter
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "simpletest",
			args: args{
				prefix: "theprefix.",
				params: []Parameter{{Name: "foo", Value: "bar"}},
			},
			want: map[string]string{"foo": "bar"},
		},
		{
			name: "simpletest with git var",
			args: args{
				prefix: "theprefix.",
				params: []Parameter{{Name: "git.hash", Value: "bar"}},
			},
			want: map[string]string{"theprefix.git.hash": "bar"},
		},
		{
			name: "simpletest with cds var",
			args: args{
				prefix: "theprefix.",
				params: []Parameter{{Name: "cds.name", Value: "bar"}},
			},
			want: map[string]string{"theprefix.cds.name": "bar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParametersToMapWithPrefixBuiltinVar(tt.args.prefix, tt.args.params); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParametersToMapWithPrefixBuiltinVar() = %v, want %v", got, tt.want)
			}
		})
	}
}
