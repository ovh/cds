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

// event.go converts termbox events to the termdash format.

import (
	"image"

	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/mouse"
	"github.com/mum4k/termdash/terminal/terminalapi"
	tbx "github.com/nsf/termbox-go"
)

// tbxToTd maps termbox key values to the termdash format.
var tbxToTd = map[tbx.Key]keyboard.Key{
	tbx.KeySpace:      keyboard.KeySpace,
	tbx.KeyF1:         keyboard.KeyF1,
	tbx.KeyF2:         keyboard.KeyF2,
	tbx.KeyF3:         keyboard.KeyF3,
	tbx.KeyF4:         keyboard.KeyF4,
	tbx.KeyF5:         keyboard.KeyF5,
	tbx.KeyF6:         keyboard.KeyF6,
	tbx.KeyF7:         keyboard.KeyF7,
	tbx.KeyF8:         keyboard.KeyF8,
	tbx.KeyF9:         keyboard.KeyF9,
	tbx.KeyF10:        keyboard.KeyF10,
	tbx.KeyF11:        keyboard.KeyF11,
	tbx.KeyF12:        keyboard.KeyF12,
	tbx.KeyInsert:     keyboard.KeyInsert,
	tbx.KeyDelete:     keyboard.KeyDelete,
	tbx.KeyHome:       keyboard.KeyHome,
	tbx.KeyEnd:        keyboard.KeyEnd,
	tbx.KeyPgup:       keyboard.KeyPgUp,
	tbx.KeyPgdn:       keyboard.KeyPgDn,
	tbx.KeyArrowUp:    keyboard.KeyArrowUp,
	tbx.KeyArrowDown:  keyboard.KeyArrowDown,
	tbx.KeyArrowLeft:  keyboard.KeyArrowLeft,
	tbx.KeyArrowRight: keyboard.KeyArrowRight,
	tbx.KeyCtrlTilde:  keyboard.KeyCtrlTilde,
	tbx.KeyCtrlA:      keyboard.KeyCtrlA,
	tbx.KeyCtrlB:      keyboard.KeyCtrlB,
	tbx.KeyCtrlC:      keyboard.KeyCtrlC,
	tbx.KeyCtrlD:      keyboard.KeyCtrlD,
	tbx.KeyCtrlE:      keyboard.KeyCtrlE,
	tbx.KeyCtrlF:      keyboard.KeyCtrlF,
	tbx.KeyCtrlG:      keyboard.KeyCtrlG,
	tbx.KeyBackspace:  keyboard.KeyBackspace,
	tbx.KeyTab:        keyboard.KeyTab,
	tbx.KeyCtrlJ:      keyboard.KeyCtrlJ,
	tbx.KeyCtrlK:      keyboard.KeyCtrlK,
	tbx.KeyCtrlL:      keyboard.KeyCtrlL,
	tbx.KeyEnter:      keyboard.KeyEnter,
	tbx.KeyCtrlN:      keyboard.KeyCtrlN,
	tbx.KeyCtrlO:      keyboard.KeyCtrlO,
	tbx.KeyCtrlP:      keyboard.KeyCtrlP,
	tbx.KeyCtrlQ:      keyboard.KeyCtrlQ,
	tbx.KeyCtrlR:      keyboard.KeyCtrlR,
	tbx.KeyCtrlS:      keyboard.KeyCtrlS,
	tbx.KeyCtrlT:      keyboard.KeyCtrlT,
	tbx.KeyCtrlU:      keyboard.KeyCtrlU,
	tbx.KeyCtrlV:      keyboard.KeyCtrlV,
	tbx.KeyCtrlW:      keyboard.KeyCtrlW,
	tbx.KeyCtrlX:      keyboard.KeyCtrlX,
	tbx.KeyCtrlY:      keyboard.KeyCtrlY,
	tbx.KeyCtrlZ:      keyboard.KeyCtrlZ,
	tbx.KeyEsc:        keyboard.KeyEsc,
	tbx.KeyCtrl4:      keyboard.KeyCtrl4,
	tbx.KeyCtrl5:      keyboard.KeyCtrl5,
	tbx.KeyCtrl6:      keyboard.KeyCtrl6,
	tbx.KeyCtrl7:      keyboard.KeyCtrl7,
	tbx.KeyBackspace2: keyboard.KeyBackspace2,
}

// convKey converts a termbox keyboard event to the termdash format.
func convKey(tbxEv tbx.Event) terminalapi.Event {
	if tbxEv.Key != 0 && tbxEv.Ch != 0 {
		return terminalapi.NewErrorf("the key event contain both a key(%v) and a character(%v)", tbxEv.Key, tbxEv.Ch)
	}

	if tbxEv.Ch != 0 {
		return &terminalapi.Keyboard{
			Key: keyboard.Key(tbxEv.Ch),
		}
	}

	k, ok := tbxToTd[tbxEv.Key]
	if !ok {
		return terminalapi.NewErrorf("unknown keyboard key '%v' in a keyboard event", k)
	}
	return &terminalapi.Keyboard{
		Key: k,
	}
}

// convMouse converts a termbox mouse event to the termdash format.
func convMouse(tbxEv tbx.Event) terminalapi.Event {
	var button mouse.Button

	switch k := tbxEv.Key; k {
	case tbx.MouseLeft:
		button = mouse.ButtonLeft
	case tbx.MouseMiddle:
		button = mouse.ButtonMiddle
	case tbx.MouseRight:
		button = mouse.ButtonRight
	case tbx.MouseRelease:
		button = mouse.ButtonRelease
	case tbx.MouseWheelUp:
		button = mouse.ButtonWheelUp
	case tbx.MouseWheelDown:
		button = mouse.ButtonWheelDown
	default:
		return terminalapi.NewErrorf("unknown mouse key %v in a mouse event", k)
	}

	return &terminalapi.Mouse{
		Position: image.Point{tbxEv.MouseX, tbxEv.MouseY},
		Button:   button,
	}
}

// convResize converts a termbox resize event to the termdash format.
func convResize(tbxEv tbx.Event) terminalapi.Event {
	size := image.Point{tbxEv.Width, tbxEv.Height}
	if size.X < 0 || size.Y < 0 {
		return terminalapi.NewErrorf("terminal resized to negative size: %v", size)
	}
	return &terminalapi.Resize{
		Size: size,
	}
}

// toTermdashEvents converts a termbox event to the termdash event format.
func toTermdashEvents(tbxEv tbx.Event) []terminalapi.Event {
	switch t := tbxEv.Type; t {
	case tbx.EventInterrupt:
		return []terminalapi.Event{
			terminalapi.NewError("event type EventInterrupt isn't supported"),
		}
	case tbx.EventRaw:
		return []terminalapi.Event{
			terminalapi.NewError("event type EventRaw isn't supported"),
		}
	case tbx.EventNone:
		return []terminalapi.Event{
			terminalapi.NewError("event type EventNone isn't supported"),
		}
	case tbx.EventError:
		return []terminalapi.Event{
			terminalapi.NewErrorf("input error occurred: %v", tbxEv.Err),
		}
	case tbx.EventResize:
		return []terminalapi.Event{convResize(tbxEv)}
	case tbx.EventMouse:
		return []terminalapi.Event{convMouse(tbxEv)}
	case tbx.EventKey:
		return []terminalapi.Event{
			convKey(tbxEv),
		}
	default:
		return []terminalapi.Event{
			terminalapi.NewErrorf("unknown termbox event type: %v", t),
		}
	}
}
