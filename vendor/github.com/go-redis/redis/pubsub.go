package redis

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-redis/redis/internal"
	"github.com/go-redis/redis/internal/pool"
)

// PubSub implements Pub/Sub commands as described in
// http://redis.io/topics/pubsub. It's NOT safe for concurrent use by
// multiple goroutines.
//
// PubSub automatically resubscribes to the channels and patterns
// when Redis becomes unavailable.
type PubSub struct {
	base baseClient

	mu     sync.Mutex
	cn     *pool.Conn
	closed bool

	subMu    sync.Mutex
	channels []string
	patterns []string

	cmd *Cmd
}

func (c *PubSub) conn() (*pool.Conn, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, false, pool.ErrClosed
	}

	if c.cn != nil {
		return c.cn, false, nil
	}

	cn, err := c.base.connPool.NewConn()
	if err != nil {
		return nil, false, err
	}

	if !cn.Inited {
		if err := c.base.initConn(cn); err != nil {
			_ = c.base.connPool.CloseConn(cn)
			return nil, false, err
		}
	}

	if err := c.resubscribe(cn); err != nil {
		_ = c.base.connPool.CloseConn(cn)
		return nil, false, err
	}

	c.cn = cn
	return cn, true, nil
}

func (c *PubSub) resubscribe(cn *pool.Conn) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	var firstErr error
	if len(c.channels) > 0 {
		if err := c._subscribe(cn, "subscribe", c.channels...); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if len(c.patterns) > 0 {
		if err := c._subscribe(cn, "psubscribe", c.patterns...); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (c *PubSub) putConn(cn *pool.Conn, err error) {
	if !internal.IsBadConn(err, true) {
		return
	}

	c.mu.Lock()
	if c.cn == cn {
		_ = c.closeConn()
	}
	c.mu.Unlock()
}

func (c *PubSub) closeConn() error {
	err := c.base.connPool.CloseConn(c.cn)
	c.cn = nil
	return err
}

func (c *PubSub) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return pool.ErrClosed
	}
	c.closed = true

	if c.cn != nil {
		return c.closeConn()
	}
	return nil
}

func (c *PubSub) subscribe(redisCmd string, channels ...string) error {
	cn, isNew, err := c.conn()
	if err != nil {
		return err
	}

	if isNew {
		return nil
	}

	err = c._subscribe(cn, redisCmd, channels...)
	c.putConn(cn, err)
	return err
}

func (c *PubSub) _subscribe(cn *pool.Conn, redisCmd string, channels ...string) error {
	args := make([]interface{}, 1+len(channels))
	args[0] = redisCmd
	for i, channel := range channels {
		args[1+i] = channel
	}
	cmd := NewSliceCmd(args...)

	cn.SetWriteTimeout(c.base.opt.WriteTimeout)
	return writeCmd(cn, cmd)
}

// Subscribes the client to the specified channels. It returns
// empty subscription if there are no channels.
func (c *PubSub) Subscribe(channels ...string) error {
	c.subMu.Lock()
	c.channels = appendIfNotExists(c.channels, channels...)
	c.subMu.Unlock()
	return c.subscribe("subscribe", channels...)
}

// Subscribes the client to the given patterns. It returns
// empty subscription if there are no patterns.
func (c *PubSub) PSubscribe(patterns ...string) error {
	c.subMu.Lock()
	c.patterns = appendIfNotExists(c.patterns, patterns...)
	c.subMu.Unlock()
	return c.subscribe("psubscribe", patterns...)
}

// Unsubscribes the client from the given channels, or from all of
// them if none is given.
func (c *PubSub) Unsubscribe(channels ...string) error {
	c.subMu.Lock()
	c.channels = remove(c.channels, channels...)
	c.subMu.Unlock()
	return c.subscribe("unsubscribe", channels...)
}

// Unsubscribes the client from the given patterns, or from all of
// them if none is given.
func (c *PubSub) PUnsubscribe(patterns ...string) error {
	c.subMu.Lock()
	c.patterns = remove(c.patterns, patterns...)
	c.subMu.Unlock()
	return c.subscribe("punsubscribe", patterns...)
}

func (c *PubSub) Ping(payload ...string) error {
	args := []interface{}{"ping"}
	if len(payload) == 1 {
		args = append(args, payload[0])
	}
	cmd := NewCmd(args...)

	cn, _, err := c.conn()
	if err != nil {
		return err
	}

	cn.SetWriteTimeout(c.base.opt.WriteTimeout)
	err = writeCmd(cn, cmd)
	c.putConn(cn, err)
	return err
}

// Message received after a successful subscription to channel.
type Subscription struct {
	// Can be "subscribe", "unsubscribe", "psubscribe" or "punsubscribe".
	Kind string
	// Channel name we have subscribed to.
	Channel string
	// Number of channels we are currently subscribed to.
	Count int
}

func (m *Subscription) String() string {
	return fmt.Sprintf("%s: %s", m.Kind, m.Channel)
}

// Message received as result of a PUBLISH command issued by another client.
type Message struct {
	Channel string
	Pattern string
	Payload string
}

func (m *Message) String() string {
	return fmt.Sprintf("Message<%s: %s>", m.Channel, m.Payload)
}

// Pong received as result of a PING command issued by another client.
type Pong struct {
	Payload string
}

func (p *Pong) String() string {
	if p.Payload != "" {
		return fmt.Sprintf("Pong<%s>", p.Payload)
	}
	return "Pong"
}

func (c *PubSub) newMessage(reply interface{}) (interface{}, error) {
	switch reply := reply.(type) {
	case string:
		return &Pong{
			Payload: reply,
		}, nil
	case []interface{}:
		switch kind := reply[0].(string); kind {
		case "subscribe", "unsubscribe", "psubscribe", "punsubscribe":
			return &Subscription{
				Kind:    kind,
				Channel: reply[1].(string),
				Count:   int(reply[2].(int64)),
			}, nil
		case "message":
			return &Message{
				Channel: reply[1].(string),
				Payload: reply[2].(string),
			}, nil
		case "pmessage":
			return &Message{
				Pattern: reply[1].(string),
				Channel: reply[2].(string),
				Payload: reply[3].(string),
			}, nil
		case "pong":
			return &Pong{
				Payload: reply[1].(string),
			}, nil
		default:
			return nil, fmt.Errorf("redis: unsupported pubsub message: %q", kind)
		}
	default:
		return nil, fmt.Errorf("redis: unsupported pubsub message: %#v", reply)
	}
}

// ReceiveTimeout acts like Receive but returns an error if message
// is not received in time. This is low-level API and most clients
// should use ReceiveMessage.
func (c *PubSub) ReceiveTimeout(timeout time.Duration) (interface{}, error) {
	if c.cmd == nil {
		c.cmd = NewCmd()
	}

	cn, _, err := c.conn()
	if err != nil {
		return nil, err
	}

	cn.SetReadTimeout(timeout)
	err = c.cmd.readReply(cn)
	c.putConn(cn, err)
	if err != nil {
		return nil, err
	}

	return c.newMessage(c.cmd.Val())
}

// Receive returns a message as a Subscription, Message, Pong or error.
// See PubSub example for details. This is low-level API and most clients
// should use ReceiveMessage.
func (c *PubSub) Receive() (interface{}, error) {
	return c.ReceiveTimeout(0)
}

// ReceiveMessage returns a Message or error ignoring Subscription or Pong
// messages. It automatically reconnects to Redis Server and resubscribes
// to channels in case of network errors.
func (c *PubSub) ReceiveMessage() (*Message, error) {
	return c.receiveMessage(5 * time.Second)
}

func (c *PubSub) receiveMessage(timeout time.Duration) (*Message, error) {
	var errNum uint
	for {
		msgi, err := c.ReceiveTimeout(timeout)
		if err != nil {
			if !internal.IsNetworkError(err) {
				return nil, err
			}

			errNum++
			if errNum < 3 {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					err := c.Ping()
					if err != nil {
						internal.Logf("PubSub.Ping failed: %s", err)
					}
				}
			} else {
				// 3 consequent errors - connection is broken or
				// Redis Server is down.
				// Sleep to not exceed max number of open connections.
				time.Sleep(time.Second)
			}
			continue
		}

		// Reset error number, because we received a message.
		errNum = 0

		switch msg := msgi.(type) {
		case *Subscription:
			// Ignore.
		case *Pong:
			// Ignore.
		case *Message:
			return msg, nil
		default:
			return nil, fmt.Errorf("redis: unknown message: %T", msgi)
		}
	}
}

// Channel returns a channel for concurrently receiving messages.
// The channel is closed with PubSub.
func (c *PubSub) Channel() <-chan *Message {
	ch := make(chan *Message, 100)
	go func() {
		for {
			msg, err := c.ReceiveMessage()
			if err != nil {
				if err == pool.ErrClosed {
					break
				}
				continue
			}
			ch <- msg
		}
		close(ch)
	}()
	return ch
}

func appendIfNotExists(ss []string, es ...string) []string {
loop:
	for _, e := range es {
		for _, s := range ss {
			if s == e {
				continue loop
			}
		}
		ss = append(ss, e)
	}
	return ss
}

func remove(ss []string, es ...string) []string {
	if len(es) == 0 {
		return ss[:0]
	}
	for _, e := range es {
		for i, s := range ss {
			if s == e {
				ss = append(ss[:i], ss[i+1:]...)
				break
			}
		}
	}
	return ss
}
