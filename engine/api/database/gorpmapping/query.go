package gorpmapping

import (
	"github.com/ovh/cds/engine/gorpmapper"
)

type Query struct {
	gorpmapper.Query
}

// NewQuery returns a new query from given string request.
func NewQuery(q string) Query { return Query{gorpmapper.NewQuery(q)} }

// Args store query arguments.
func (q Query) Args(as ...interface{}) Query {
	q.Query.Arguments = as
	return q
}

func (q Query) Limit(i int) Query {
	q.Query.Limit(i)
	return q
}
