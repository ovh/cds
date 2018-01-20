/*
Copyright 2017 The go-marathon Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package marathon

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentVariableUnmarshal(t *testing.T) {
	defaultConfig := NewDefaultConfig()
	configs := &configContainer{
		client: &defaultConfig,
		server: &serverConfig{
			scope: "environment-variables",
		},
	}

	endpoint := newFakeMarathonEndpoint(t, configs)
	defer endpoint.Close()

	application, err := endpoint.Client.Application(fakeAppName)
	require.NoError(t, err)

	env := application.Env
	secrets := application.Secrets

	require.NotNil(t, env)
	assert.Equal(t, "bar", (*env)["FOO"])
	assert.Equal(t, "TOP", (*secrets)["secret"].EnvVar)
	assert.Equal(t, "/path/to/secret", (*secrets)["secret"].Source)
}

func TestMalformedPayloadUnmarshal(t *testing.T) {
	var tests = []struct {
		expected    string
		given       []byte
		description string
	}{
		{
			expected:    "unexpected secret field",
			given:       []byte(`{"env": {"FOO": "bar", "SECRET": {"not_secret": "secret1"}}, "secrets": {"secret1": {"source": "/path/to/secret"}}}`),
			description: "Field in environment secret not equal to secret.",
		},
		{
			expected:    "unexpected secret field",
			given:       []byte(`{"env": {"FOO": "bar", "SECRET": {"secret": 1}}, "secrets": {"secret1": {"source": "/path/to/secret"}}}`),
			description: "Invalid value in environment secret.",
		},
		{
			expected:    "unexpected environment variable type",
			given:       []byte(`{"env": {"FOO": 1, "SECRET": {"secret": "secret1"}}, "secrets": {"secret1": {"source": "/path/to/secret"}}}`),
			description: "Invalid environment variable type.",
		},
		{
			expected:    "malformed application definition",
			given:       []byte(`{"env": "value"}`),
			description: "Bad application definition.",
		},
	}

	for _, test := range tests {
		tmpApp := new(Application)

		err := json.Unmarshal(test.given, &tmpApp)
		if assert.Error(t, err, test.description) {
			assert.True(t, strings.HasPrefix(err.Error(), test.expected), test.description)
		}
	}
}

func TestEnvironmentVariableMarshal(t *testing.T) {
	testApp := new(Application)
	targetString := []byte(`{"ports":null,"dependencies":null,"env":{"FOO":"bar","TOP":{"secret":"secret1"}},"secrets":{"secret1":{"source":"/path/to/secret"}}}`)
	testApp.AddEnv("FOO", "bar")
	testApp.AddSecret("TOP", "secret1", "/path/to/secret")

	app, err := json.Marshal(testApp)
	if assert.NoError(t, err) {
		assert.Equal(t, targetString, app)
	}
}
