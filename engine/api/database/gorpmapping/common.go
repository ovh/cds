package gorpmapping

import (
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"

	"github.com/ovh/cds/sdk"
)

const (
	// ViolateUniqueKeyPGCode is the pg code when duplicating unique key
	ViolateUniqueKeyPGCode = "23505"
)

// IDsToQueryString returns a comma separated list of given ids.
func IDsToQueryString(ids []int64) string {
	res := make([]string, len(ids))
	for i := range ids {
		res[i] = fmt.Sprintf("%d", ids[i])
	}
	return strings.Join(res, ",")
}

// Insert value in given db.
func Insert(db gorp.SqlExecutor, i interface{}) error {
	err := db.Insert(i)
	if e, ok := err.(*pq.Error); ok && e.Code == ViolateUniqueKeyPGCode {
		err = sdk.NewError(sdk.ErrConflict, e)
	}
	return sdk.WithStack(err)
}

// Update value in given db.
func Update(db gorp.SqlExecutor, i interface{}) error {
	_, err := db.Update(i)
	return sdk.WithStack(err)
}

// Delete value in given db.
func Delete(db gorp.SqlExecutor, i interface{}) error {
	_, err := db.Delete(i)
	return sdk.WithStack(err)
}

// And returns givens query strings joined with AND.
func And(qs ...string) string { return fmt.Sprintf("(%s)", strings.Join(qs, " AND ")) }
