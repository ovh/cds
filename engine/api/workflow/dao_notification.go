package workflow

import (
	"database/sql"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

func DeleteNotifications(db gorp.SqlExecutor, workflowID int64) error {
	_, err := db.Exec("DELETE FROM workflow_notification where workflow_id = $1", workflowID)
	if err != nil {
		return sdk.WrapError(err, "Cannot delete notifications on workflow %d", workflowID)
	}
	return nil
}

func loadNotifications(db gorp.SqlExecutor, w *sdk.Workflow) ([]sdk.WorkflowNotification, error) {
	notifIDs := []int64{}
	_, err := db.Select(&notifIDs, "select id from workflow_notification where workflow_id = $1", w.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, sdk.WrapError(err, "Unable to load notification IDs on workflow %d", w.ID)
	}

	notifications := make([]sdk.WorkflowNotification, len(notifIDs))
	for index, id := range notifIDs {
		n, errJ := loadNotification(db, w, id)
		if errJ != nil {
			return nil, sdk.WrapError(errJ, "loadNotification> Unable to load notification %d on workflow %d", id, w.ID)
		}
		notifications[index] = n
	}

	return notifications, nil
}

func loadNotification(db gorp.SqlExecutor, w *sdk.Workflow, id int64) (sdk.WorkflowNotification, error) {
	dbnotif := Notification{}
	//Load the notification
	if err := db.SelectOne(&dbnotif, "select * from workflow_notification where id = $1", id); err != nil {
		return sdk.WorkflowNotification{}, sdk.WrapError(err, "Unable to load notification %d", id)
	}
	dbnotif.WorkflowID = w.ID

	//Load sources
	var ids []int64
	if _, err := db.Select(&ids, "select node_id from workflow_notification_source where workflow_notification_id = $1", id); err != nil {
		return sdk.WorkflowNotification{}, sdk.WrapError(err, "Unable to load notification %d sources", id)
	}
	dbnotif.NodeIDs = ids
	n := sdk.WorkflowNotification(dbnotif)

	for _, id := range n.NodeIDs {
		notifNode := w.WorkflowData.NodeByID(id)
		if notifNode != nil {
			n.SourceNodeRefs = append(n.SourceNodeRefs, notifNode.Name)
		}
	}

	return n, nil
}

func loadVCSNotificationWithNodeID(db gorp.SqlExecutor, workflowID, nodeID int64) (sdk.WorkflowNotification, error) {
	dbnotif := Notification{}
	query := `SELECT workflow_notification.*
	FROM workflow_notification
		JOIN workflow_notification_source ON workflow_notification.id = workflow_notification_source.workflow_notification_id
	WHERE workflow_notification.workflow_id = $1 AND workflow_notification_source.node_id = $2 AND workflow_notification.type = $3`
	//Load the notification
	if err := db.SelectOne(&dbnotif, query, workflowID, nodeID, sdk.VCSUserNotification); err != nil {
		if err == sql.ErrNoRows {
			return sdk.WorkflowNotification{}, nil
		}
		return sdk.WorkflowNotification{}, sdk.WrapError(err, "Unable to load notification for workflow id %d and node id %d", workflowID, nodeID)
	}
	dbnotif.WorkflowID = workflowID

	return sdk.WorkflowNotification(dbnotif), nil
}

func InsertNotification(db gorp.SqlExecutor, w *sdk.Workflow, n *sdk.WorkflowNotification) error {
	n.WorkflowID = w.ID
	n.ID = 0
	n.NodeIDs = nil
	dbNotif := Notification(*n)

	//Check references to sources
	if len(n.SourceNodeRefs) == 0 {
		return sdk.WrapError(sdk.ErrWorkflowNodeRef, "insertNotification> No notification references")
	}

	for _, s := range n.SourceNodeRefs {
		nodeFoundRef := w.WorkflowData.NodeByName(s)
		if nodeFoundRef == nil || nodeFoundRef.ID == 0 {
			return sdk.WrapError(sdk.ErrWorkflowNotificationNodeRef, "insertNotification> Invalid notification references node %s", s)
		}
		n.NodeIDs = append(n.NodeIDs, nodeFoundRef.ID)
	}

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
