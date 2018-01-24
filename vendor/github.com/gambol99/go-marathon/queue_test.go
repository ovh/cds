/*
Copyright 2016 The go-marathon Authors All rights reserved.

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

func TestQueue(t *testing.T) {
	endpoint := newFakeMarathonEndpoint(t, nil)
	defer endpoint.Close()

	queue, err := endpoint.Client.Queue()
	assert.NoError(t, err)
	assert.NotNil(t, queue)

	assert.Len(t, queue.Items, 1)
	item := queue.Items[0]
	assert.Equal(t, item.Count, 10)
	assert.Equal(t, item.Delay.Overdue, true)
	assert.Equal(t, item.Delay.TimeLeftSeconds, 784)
	assert.NotEmpty(t, item.Application.ID)
}

func TestDeleteQueueDelay(t *testing.T) {
	endpoint := newFakeMarathonEndpoint(t, nil)
	defer endpoint.Close()

	err := endpoint.Client.DeleteQueueDelay(fakeAppName)
	assert.NoError(t, err)
}
