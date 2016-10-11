package ramsql

import (
	"fmt"
)

// Tx implements SQL transaction method
type Tx struct {
	conn *Conn
}

// Commit the transaction on server
func (t *Tx) Commit() error {
	// TODO: Not implemented
	return nil
}

// Rollback all changes
func (t *Tx) Rollback() error {
	return fmt.Errorf("Not implemented")
}
