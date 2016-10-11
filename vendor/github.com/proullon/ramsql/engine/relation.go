package engine

import (
	"sync"
)

// Relation is a table with column and rows
// AKA File
type Relation struct {
	sync.RWMutex
	table *Table
	rows  []*Tuple
}

// NewRelation initializes a new Relation struct
func NewRelation(t *Table) *Relation {
	r := &Relation{
		table: t,
	}

	return r
}

// Insert a tuple in relation
func (r *Relation) Insert(t *Tuple) error {
	// Maybe do somthing like lock read/write here
	// Maybe index
	r.rows = append(r.rows, t)
	return nil
}
