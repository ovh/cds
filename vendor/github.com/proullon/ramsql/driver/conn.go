package ramsql

import (
	"database/sql/driver"
	"sync"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/protocol"
)

// Conn implements sql/driver Conn interface
type Conn struct {
	// Mutex is locked when a Statement is created
	// then released on Statement.Exec or Statement.Query
	mutex sync.Mutex

	// Socket is the network connection to RamSQL engine
	conn protocol.DriverConn
	// socket net.Conn

	// This conn belongs to this server
	parent *Server
}

func newConn(conn protocol.DriverConn, parent *Server) driver.Conn {
	parent.openingConn()
	return &Conn{conn: conn, parent: parent}
}

// Prepare returns a prepared statement, bound to this connection.
func (c *Conn) Prepare(query string) (driver.Stmt, error) {

	stmt := prepareStatement(c, query)

	return stmt, nil
}

// Close invalidates and potentially stops any current
// prepared statements and transactions, marking this
// connection as no longer in use.
//
// Because the sql package maintains a free pool of
// connections and only calls Close when there's a surplus of
// idle connections, it shouldn't be necessary for drivers to
// do their own connection caching.
func (c *Conn) Close() error {
	log.Debug("Conn.Close")
	c.conn.Close()

	if c.parent != nil {
		c.parent.closingConn()
	}

	return nil
}

// Begin starts and returns a new transaction.
func (c *Conn) Begin() (driver.Tx, error) {

	tx := Tx{
		conn: c,
	}

	return &tx, nil
}
