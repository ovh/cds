package action

import (
	"encoding/json"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

// LoadAuditAction loads from database the last 10 versions of an action definition
func LoadAuditAction(db gorp.SqlExecutor, actionID int, public bool) ([]sdk.ActionAudit, error) {
	audits := []sdk.ActionAudit{}
	query := `
		SELECT
			action_audit.change, action_audit.versionned, action_audit.action_json,
			"user".username
		FROM action_audit
		JOIN "user" ON "user".id = action_audit.user_id
		JOIN action ON action.id = action_audit.action_id
		WHERE action_audit.action_id = $1 AND action.public = $2
		ORDER by action_audit.versionned DESC
	`
	rows, err := db.Query(query, actionID, public)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var audit sdk.ActionAudit
		var actionData string
		err = rows.Scan(&audit.Change, &audit.Versionned, &actionData, &audit.User.Username)
		if err != nil {
			return nil, err
		}

		a := &sdk.Action{}
		if err := json.Unmarshal([]byte(actionData), a); err != nil {
			return nil, err
		}

		audit.Action = *a
		audits = append(audits, audit)
	}
	return audits, nil
}
