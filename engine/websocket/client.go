package websocket

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rockbears/log"
	"github.com/tevino/abool"

	"github.com/ovh/cds/sdk"
)

const (
	// writeTimeout bounds a single write to a client. Without it a client that stopped reading holds
	// the goroutine writing to it for as long as it likes. A connection whose write deadline was
	// exceeded is left in an undefined state, so such a client is dropped rather than written to
	// again.
	writeTimeout = 10 * time.Second
	// maxPendingWrites bounds the writes waiting their turn on one client. Each of them holds a
	// goroutine and the message it carries until the client reads, so a client falling behind what it
	// subscribed to is dropped instead of being waited for.
	maxPendingWrites = 64
)

func NewClient(conn *websocket.Conn) Client {
	return &CommonClient{
		uuid:     sdk.UUID(),
		conn:     conn,
		isClosed: abool.NewBool(false),
	}
}

type Client interface {
	UUID() string
	Listen(context.Context, *sdk.GoRoutines) error
	OnMessage(func([]byte))
	Send(interface{}) error
	Close()
}

type CommonClient struct {
	uuid      string
	mutex     sync.Mutex
	pending   atomic.Int64
	conn      *websocket.Conn
	isClosed  *abool.AtomicBool
	onMessage func([]byte)
}

func (c *CommonClient) UUID() string { return c.uuid }

func (c *CommonClient) OnMessage(f func([]byte)) { c.onMessage = f }

func (c *CommonClient) Send(m interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = sdk.WithStack(fmt.Errorf("websocketClient.Send recovered %v", r))
		}
	}()

	if pending := c.pending.Add(1); pending > maxPendingWrites {
		c.pending.Add(-1)
		return sdk.WithStack(fmt.Errorf("client %s is too far behind, %d writes pending", c.uuid, pending-1))
	}
	defer c.pending.Add(-1)

	// Lock avoid parallel write on same conn
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn == nil || c.isClosed.IsSet() {
		return sdk.WithStack(fmt.Errorf("client deconnected"))
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return sdk.WithStack(err)
	}

	if err := c.conn.WriteJSON(m); err != nil {
		// A write that failed leaves the connection in an undefined state, so the error is returned
		// whatever it is: the caller drops the client rather than writing to it again.
		// ErrCloseSent is returned when the application writes a message to the connection after sending a close message.
		if err != websocket.ErrCloseSent && !strings.Contains(err.Error(), "use of closed network connection") {
			err = sdk.WrapError(err, "can't send to client %s", c.uuid)
			ctx := sdk.ContextWithStacktrace(context.Background(), err)
			log.Error(ctx, "%v", err)
		}
		return sdk.WithStack(err)
	}

	return nil
}

func (c *CommonClient) Close() { c.isClosed.Set() }

func (c *CommonClient) Listen(ctx context.Context, gorts *sdk.GoRoutines) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	inMessageChan := make(chan []byte, 10)
	defer close(inMessageChan)

	gorts.Exec(ctx, fmt.Sprintf("websocket.Client.Listen.readInMessages-%s", c.uuid), func(ctx context.Context) {
		for {
			select {
			case <-ctx.Done():
				return
			case m, more := <-inMessageChan:
				if !more {
					return
				}
				if c.onMessage != nil {
					c.onMessage(m)
				}
			}
		}
	})

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				err = sdk.WrapError(err, "websocket unexpected error occurred")
				log.Error(sdk.ContextWithStacktrace(ctx, err), "%v", err)
			}
			log.Debug(ctx, "websocket.Client.Listen> client %s disconnected", c.uuid)
			break
		}

		inMessageChan <- msg
	}

	return nil
}
