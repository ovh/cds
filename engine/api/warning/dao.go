package warning

import "github.com/go-gorp/gorp"

func removeWarning(db gorp.SqlExecutor, warningType string, element string) error {
	_, err := db.Exec("DELETE FROM warning where type = $1 and element = $2", warningType, element)
	return err
}
