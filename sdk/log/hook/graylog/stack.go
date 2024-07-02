package graylog

import "sync"

// NewStack returns a new stack.
func NewStack(cap int) *Stack {
	s := &Stack{
		size:     cap,
		cap:      cap,
		messages: make([]Message, cap),
	}
	return s
}

// Stack is a basic LIFO stack that resizes as needed.
type Stack struct {
	mutex    sync.Mutex
	messages []Message
	size     int
	head     int
	tail     int
	count    int
	cap      int
}

// Push adds a node to the queue.
func (q *Stack) Push(n Message) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.head == q.tail && q.count > 0 {
		messages := make([]Message, len(q.messages)+q.size)
		copy(messages, q.messages[q.head:])
		copy(messages[len(q.messages)-q.head:], q.messages[:q.head])
		q.head = 0
		q.tail = len(q.messages)
		q.messages = messages
	}
	q.messages[q.tail] = n
	q.tail = (q.tail + 1) % len(q.messages)
	q.count++
}

func (q *Stack) Ready() bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	return q.count < q.cap
}

func (q *Stack) Empty() bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	return q.count == 0
}

// Pop removes and returns a node from the queue in first to last order.
func (q *Stack) Pop() (Message, bool) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.count == 0 {
		return Message{}, false
	}
	node := q.messages[q.head]
	q.head = (q.head + 1) % len(q.messages)
	q.count--
	return node, true
}
