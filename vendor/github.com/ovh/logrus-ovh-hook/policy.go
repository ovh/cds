package logrus_ovh

// DropPolicy will drop the message if the channel is full
func DropPolicy(msg *Message, ch chan *Message) {
	select {
	case ch <- msg:
	default:
	}
}
