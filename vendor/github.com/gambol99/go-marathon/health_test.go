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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	hc := new(HealthCheck)
	command := Command{"curl localhost:8080"}
	hc.SetCommand(command)
	assert.Equal(t, command, (*hc.Command))
}

func TestPortIndex(t *testing.T) {
	hc := new(HealthCheck)
	hc.SetPortIndex(0)
	assert.Equal(t, 0, (*hc.PortIndex))
}

func TestPort(t *testing.T) {
	hc := new(HealthCheck)
	hc.SetPort(8000)
	assert.Equal(t, 8000, (*hc.Port))
}

func TestPath(t *testing.T) {
	hc := new(HealthCheck)
	hc.SetPath("/path")
	assert.Equal(t, "/path", (*hc.Path))
}

func TestMaxConsecutiveFailures(t *testing.T) {
	hc := new(HealthCheck)
	hc.SetMaxConsecutiveFailures(3)
	assert.Equal(t, 3, (*hc.MaxConsecutiveFailures))
}

func TestIgnoreHTTP1xx(t *testing.T) {
	hc := new(HealthCheck)
	hc.SetIgnoreHTTP1xx(true)
	assert.True(t, (*hc.IgnoreHTTP1xx))
}
