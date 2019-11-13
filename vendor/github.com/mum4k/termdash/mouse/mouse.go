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

// Package mouse defines known mouse buttons.
package mouse

// Button represents a mouse button.
type Button int

// String implements fmt.Stringer()
func (b Button) String() string {
	if n, ok := buttonNames[b]; ok {
		return n
	}
	return "ButtonUnknown"
}

// buttonNames maps Button values to human readable names.
var buttonNames = map[Button]string{
	ButtonLeft:      "ButtonLeft",
	ButtonRight:     "ButtonRight",
	ButtonMiddle:    "ButtonMiddle",
	ButtonRelease:   "ButtonRelease",
	ButtonWheelUp:   "ButtonWheelUp",
	ButtonWheelDown: "ButtonWheelDown",
}

// Buttons recognized on the mouse.
const (
	buttonUnknown Button = iota
	ButtonLeft
	ButtonRight
	ButtonMiddle
	ButtonRelease
	ButtonWheelUp
	ButtonWheelDown
)
