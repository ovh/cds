// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.
// inspired from github.com/gemnasium/logrus-graylog-hook

package hook

// DropPolicy will drop the message if the channel is full
func DropPolicy(msg *Message, ch chan *Message) {
	select {
	case ch <- msg:
	default:
	}
}
