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
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReadinessCheck(t *testing.T) {
	rc := ReadinessCheck{}
	rc.SetName("readiness").
		SetProtocol("HTTP").
		SetPath("/ready").
		SetPortName("http").
		SetInterval(3 * time.Second).
		SetTimeout(5 * time.Second).
		SetHTTPStatusCodesForReady([]int{200, 201}).
		SetPreserveLastResponse(true)

	if assert.NotNil(t, rc.Name) {
		assert.Equal(t, "readiness", *rc.Name)
	}
	assert.Equal(t, rc.Protocol, "HTTP")
	assert.Equal(t, rc.Path, "/ready")
	assert.Equal(t, rc.PortName, "http")
	assert.Equal(t, rc.IntervalSeconds, 3)
	assert.Equal(t, rc.TimeoutSeconds, 5)
	if assert.NotNil(t, rc.HTTPStatusCodesForReady) {
		assert.Equal(t, *rc.HTTPStatusCodesForReady, []int{200, 201})
	}
	if assert.NotNil(t, rc.PreserveLastResponse) {
		assert.True(t, *rc.PreserveLastResponse)
	}
}
