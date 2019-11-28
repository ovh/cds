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

package termbox

import (
	"fmt"

	"github.com/mum4k/termdash/terminal/terminalapi"
	tbx "github.com/nsf/termbox-go"
)

// colorMode converts termdash color modes to the termbox format.
func colorMode(cm terminalapi.ColorMode) (tbx.OutputMode, error) {
	switch cm {
	case terminalapi.ColorModeNormal:
		return tbx.OutputNormal, nil
	case terminalapi.ColorMode256:
		return tbx.Output256, nil
	case terminalapi.ColorMode216:
		return tbx.Output216, nil
	case terminalapi.ColorModeGrayscale:
		return tbx.OutputGrayscale, nil
	default:
		return -1, fmt.Errorf("don't know how to convert color mode %v to the termbox format", cm)
	}
}
