package gorpmapping

import (
	"database/sql"
	"encoding/json"

	"github.com/ovh/cds/sdk"
)

//JSONToNullString returns a valid sql.NullString with json-marshalled i
func JSONToNullString(i interface{}) (sql.NullString, error) {
	if i == nil {
		return sql.NullString{Valid: false}, nil
	}
	b, err := json.Marshal(i)
	if err != nil {
		return sql.NullString{Valid: false}, err
	}
	return sql.NullString{Valid: true, String: string(b)}, nil
}

//JSONNullString sets the holder with unmarshalled sql.NullString
func JSONNullString(s sql.NullString, holder interface{}) error {
	if !s.Valid {
		return nil
	}
	return sdk.JSONUnmarshal([]byte(s.String), holder)
}
