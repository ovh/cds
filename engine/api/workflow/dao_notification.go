package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/gorpmapping"
)

func DeleteNotifications(db gorp.SqlExecutor, workflowID int64) error {
	_, err := db.Exec("DELETE FROM workflow_notification where workflow_id = $1", workflowID)
	if err != nil {
		return sdk.WrapError(err, "Cannot delete notifications on workflow %d", workflowID)
	}
	return nil
}

func LoadNotificationsByWorkflowIDs(db gorp.SqlExecutor, ids []int64) (map[int64][]sdk.WorkflowNotification, error) {
	query := `
		SELECT 
		workflow_notification.*,
		array_remove(array_agg(workflow_notification_source.node_id::text), NULL)  "node_ids"
		FROM workflow_notification
		LEFT OUTER JOIN workflow_notification_source ON workflow_notification_source.workflow_notification_id = workflow_notification.id
		WHERE workflow_notification.workflow_id = ANY($1)
		GROUP BY workflow_notification.workflow_id, workflow_notification.id
		ORDER BY workflow_notification.workflow_id`

	var dbNotifs = []struct {
		ID         int64                        `db:"id"`
		WorkflowID int64                        `db:"workflow_id"`
		NodeIDs    pq.Int64Array                `db:"node_ids"`
		Type       string                       `db:"type"`
		Settings   sdk.UserNotificationSettings `db:"settings"`
	}{}

	if _, err := db.Select(&dbNotifs, query, pq.Int64Array(ids)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WithStack(err)
	}

	mapNotifs := make(map[int64][]sdk.WorkflowNotification)

	for _, n := range dbNotifs {
		arrayNotif := mapNotifs[n.WorkflowID]
		notif := sdk.WorkflowNotification{
			ID:         n.ID,
			Settings:   n.Settings,
			NodeIDs:    n.NodeIDs,
			Type:       n.Type,
			WorkflowID: n.WorkflowID,
		}
		// Need the node_name for references...
		arrayNotif = append(arrayNotif, notif)
		mapNotifs[n.WorkflowID] = arrayNotif
	}

	return mapNotifs, nil
}

func InsertNotification(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNotification) error {
	n.WorkflowID = w.ID
	n.ID = 0
	n.NodeIDs = nil

	for _, s := range n.SourceNodeRefs {
		nodeFoundRef := w.WorkflowData.NodeByName(s)
		if nodeFoundRef == nil || nodeFoundRef.ID == 0 {
			return sdk.WrapError(sdk.ErrWorkflowNotificationNodeRef, "insertNotification> Invalid notification references node %s", s)
		}
		n.NodeIDs = append(n.NodeIDs, nodeFoundRef.ID)
	}

	dbNotif := Notification(*n)

	//Insert the notification
	if err := db.Insert(&dbNotif); err != nil {
		return sdk.WrapError(err, "Unable to insert workflow notification")
	}
	n.ID = dbNotif.ID

	//Insert associations with sources
	query := "insert into workflow_notification_source(workflow_notification_id, node_id) values ($1, $2)"
	for i := range n.NodeIDs {
		if _, err := db.Exec(query, n.ID, n.NodeIDs[i]); err != nil {
			return sdk.WrapError(err, "Unable to insert associations between node %d and notification %d", n.NodeIDs[i], n.ID)
		}
	}

	return nil
}

// PostInsert is a db hook
func (no *Notification) PostInsert(db gorp.SqlExecutor) error {
	b, err := gorpmapping.JSONToNullString(no.Settings)
	if err != nil {
		return err
	}
	if _, err := db.Exec("update workflow_notification set settings = $1 where id = $2", b, no.ID); err != nil {
		return err
	}
	return nil
}

// PostGet is a db hook
func (no *Notification) PostGet(db gorp.SqlExecutor) error {
	res, err := db.SelectNullStr("SELECT settings FROM workflow_notification WHERE id = $1", no.ID)
	if err != nil {
		return sdk.WrapError(err, "Unable to load marshalled workflow notification")
	}

	if err := gorpmapping.JSONNullString(res, &no.Settings); err != nil {
		return sdk.WrapError(err, "cannot parse user notification")
	}

	return nil
}
