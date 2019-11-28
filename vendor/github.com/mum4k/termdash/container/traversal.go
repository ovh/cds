// Copyright 2018 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package container

import (
	"errors"
	"fmt"
)

// traversal.go provides functions that navigate the container tree.

// rootCont returns the root container.
func rootCont(c *Container) *Container {
	for p := c.parent; p != nil; p = c.parent {
		c = p
	}
	return c
}

// visitFunc is executed during traversals when node is visited.
// If the visit function returns an error, the traversal terminates and the
// errStr is set to the text of the returned error.
type visitFunc func(*Container) error

// preOrder performs pre-order DFS traversal on the container tree.
func preOrder(c *Container, errStr *string, visit visitFunc) {
	if c == nil || *errStr != "" {
		return
	}

	if err := visit(c); err != nil {
		*errStr = err.Error()
		return
	}
	preOrder(c.first, errStr, visit)
	preOrder(c.second, errStr, visit)
}

// postOrder performs post-order DFS traversal on the container tree.
func postOrder(c *Container, errStr *string, visit visitFunc) {
	if c == nil || *errStr != "" {
		return
	}

	postOrder(c.first, errStr, visit)
	postOrder(c.second, errStr, visit)
	if err := visit(c); err != nil {
		*errStr = err.Error()
		return
	}
}

// findID finds container with the provided ID.
// Returns an error of there is no container with the specified ID.
func findID(root *Container, id string) (*Container, error) {
	if id == "" {
		return nil, errors.New("the container ID must not be empty")
	}

	var (
		errStr string
		cont   *Container
	)
	preOrder(root, &errStr, visitFunc(func(c *Container) error {
		if c.opts.id == id {
			cont = c
		}
		return nil
	}))
	if cont == nil {
		return nil, fmt.Errorf("cannot find container with ID %q", id)
	}
	return cont, nil
}
