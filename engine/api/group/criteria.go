package group

import (
	"github.com/ovh/cds/engine/api/database"
)

func NewCriteria() Criteria { return Criteria{} }

type Criteria struct {
	ids []int64
}

func (c Criteria) IDs(ids ...int64) Criteria {
	c.ids = ids
	return c
}

func (c Criteria) where() string {
	var reqs []string
	if c.ids != nil {
		reqs = append(reqs, "id = ANY(string_to_array(:ids, ',')::int[])")
	}

	if len(reqs) == 0 {
		return "false"
	}

	return database.And(reqs...)
}

func (c Criteria) args() interface{} {
	return map[string]interface{}{
		"ids": database.IDsToQueryString(c.ids),
	}
}
