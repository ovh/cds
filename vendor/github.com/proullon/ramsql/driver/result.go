package ramsql

// Result is the type returned by sql/driver after an Exec statement.
type Result struct {
	err            error
	lastInsertedID int64
	rowsAffected   int64
}

func newResult(lastInsertedID int64, rowsAffected int64) *Result {
	r := &Result{
		lastInsertedID: lastInsertedID,
		rowsAffected:   rowsAffected,
	}

	return r
}

// LastInsertId returns the database's auto-generated ID
// after, for example, an INSERT into a table with primary
// key.
func (r *Result) LastInsertId() (int64, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.lastInsertedID, nil
}

// RowsAffected returns the number of rows affected by the
// query.
func (r *Result) RowsAffected() (int64, error) {
	if r.err != nil {
		return 0, r.err
	}
	return r.rowsAffected, nil
}
