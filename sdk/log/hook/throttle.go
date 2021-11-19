package hook

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type ThrottlePolicyConfig struct {
	Amount int
	Period time.Duration
	Policy ThrottlePolicy
}

type ThrottlePolicy interface {
	Init(hook *Hook)
	HandleTrailingMessage(m Message)
	PendingTrailingMessages() bool
	Flush()
}

func NewDefaultThrottlePolicy() ThrottlePolicy {
	o := &DefaultThrottlePolicy{}
	return o
}

var _ ThrottlePolicy = new(DefaultThrottlePolicy)

type DefaultThrottlePolicy struct {
	mutex  sync.Mutex
	buffer chan Message
}

func (d *DefaultThrottlePolicy) Init(hook *Hook) {
	d.buffer = make(chan Message, BufSize)
	go func() {
		for {
			time.Sleep(1 * time.Millisecond)
			if !hook.IsThrottled() {
				m, more := <-d.buffer
				hook.throttleStack.Push(m)
				if !more {
					fmt.Fprintf(os.Stderr, "[graylog] exiting trailing message goroutine...\n")
					break
				}
			}
		}
	}()
}

func (d *DefaultThrottlePolicy) HandleTrailingMessage(m Message) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	select {
	case d.buffer <- m:
	default:
		fmt.Fprintf(os.Stderr, "[graylog] message dropped\n")
	}
}

func (d *DefaultThrottlePolicy) Flush() {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for len(d.buffer) != 0 {
		time.Sleep(time.Second)
	}
}

func (d *DefaultThrottlePolicy) PendingTrailingMessages() bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	return len(d.buffer) > 0
}
