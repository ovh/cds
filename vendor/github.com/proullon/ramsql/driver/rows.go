package ramsql

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/proullon/ramsql/engine/log"
	"github.com/proullon/ramsql/engine/parser"
)

// Rows implements the sql/driver Rows interface
type Rows struct {
	rowsChannel chan []string
	columns     []string

	sync.Mutex
}

func newRows(channel chan []string) *Rows {
	r := &Rows{rowsChannel: channel}
	c, ok := <-channel
	if !ok {
		log.Critical("Cannot receive column names form channel")
		return nil
	}

	r.columns = c
	return r
}

// Columns returns the names of the columns. The number of
// columns of the result is inferred from the length of the
// slice.  If a particular column name isn't known, an empty
// string should be returned for that entry.
func (r *Rows) Columns() []string {
	return r.columns
}

// Close closes the rows iterator.
func (r *Rows) Close() error {
	r.Lock()
	defer r.Unlock()

	if r.rowsChannel == nil {
		return nil
	}

	_, ok := <-r.rowsChannel
	if !ok {
		return nil
	}

	// Tels UnlimitedRowsChannel to close itself
	//r.rowsChannel <- []string{}
	r.rowsChannel = nil
	return nil
}

// Next is called to populate the next row of data into
// the provided slice. The provided slice will be the same
// size as the Columns() are wide.
//
// The dest slice may be populated only with
// a driver Value type, but excluding string.
// All string values must be converted to []byte.
//
// Next should return io.EOF when there are no more rows.
func (r *Rows) Next(dest []driver.Value) (err error) {
	r.Lock()
	defer r.Unlock()

	if r.rowsChannel == nil {
		return io.EOF
	}

	value, ok := <-r.rowsChannel
	if !ok {
		r.rowsChannel = nil
		return io.EOF
	}

	if len(dest) < len(value) {
		return fmt.Errorf("slice too short (%d slots for %d values)", len(dest), len(value))
	}

	for i, v := range value {
		if v == "<nil>" {
			dest[i] = nil
			continue
		}

		// TODO: make rowsChannel send virtualRows,
		// so we have the type and don't blindy try to parse date here
		if t, err := parser.ParseDate(string(v)); err == nil {
			dest[i] = *t
		} else {

			dest[i] = []byte(v)
		}
	}

	return nil
}

func (r *Rows) setColumns(columns []string) {
	r.columns = columns
}

func assignValue(s string, v driver.Value) error {
	dest, ok := v.(*string)
	if !ok {
		err := errors.New("cannot assign value")
		return err
	}

	*dest = s
	return nil
}
