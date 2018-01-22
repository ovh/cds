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

func TestResidency(t *testing.T) {
	app := NewDockerApplication()

	app = app.SetResidency(TaskLostBehaviorTypeWaitForever)

	if assert.NotNil(t, app.Residency) {
		res := app.Residency

		assert.Equal(t, res.TaskLostBehavior, TaskLostBehaviorTypeWaitForever)

		res.SetRelaunchEscalationTimeout(2525 * time.Millisecond)
		// should be trimmed to seconds precision
		assert.Equal(t, app.Residency.RelaunchEscalationTimeoutSeconds, 2)

		res.SetTaskLostBehavior(TaskLostBehaviorTypeRelaunchAfterTimeout)
		assert.Equal(t, res.TaskLostBehavior, TaskLostBehaviorTypeRelaunchAfterTimeout)
	}

	app = app.EmptyResidency()

	if assert.NotNil(t, app.Residency) {
		assert.Equal(t, app.Residency, &Residency{})
	}
}
