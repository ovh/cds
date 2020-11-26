package websocket

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/tevino/abool"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
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

	// Lock avoid parallel write on same conn
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn == nil || c.isClosed.IsSet() {
		return sdk.WithStack(fmt.Errorf("client deconnected"))
	}

	if err := c.conn.WriteJSON(m); err != nil {
		// ErrCloseSent is returned when the application writes a message to the connection after sending a close message.
		if err == websocket.ErrCloseSent || strings.Contains(err.Error(), "use of closed network connection") {
			return sdk.WithStack(err)
		}
		err = sdk.WrapError(err, "can't send to client %s", c.uuid)
		log.ErrorWithFields(context.Background(), log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
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
				log.Debug("websocket.Client.Listen> read in messages routine context done")
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
				err = sdk.WrapError(err, "websocket unexpected error occured")
				log.InfoWithFields(ctx, log.Fields{"stack_trace": fmt.Sprintf("%+v", err)}, "%s", err)
			}
			log.Debug("websocket.Client.Listen> client %s disconnected", c.uuid)
			break
		}

		inMessageChan <- msg
	}

	return nil
}
