package workflow

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
)

func deleteNotifications(db gorp.SqlExecutor, workflowID int64) error {
	_, err := db.Exec("DELETE FROM workflow_notification where workflow_id = $1", workflowID)
	if err != nil {
		return sdk.WrapError(err, "deleteNotification> Cannot delete notifications on workflow %d", workflowID)
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
		return nil, sdk.WrapError(err, "loadNotification> Unable to load notification IDs on workflow %d", w.ID)
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
		return sdk.WorkflowNotification{}, sdk.WrapError(err, "loadNotification> Unable to load notification %d", id)
	}
	dbnotif.WorkflowID = w.ID

	//Load sources
	if _, err := db.Select(&dbnotif.SourceNodeIDs, "select workflow_node_id from workflow_notification_source where workflow_notification_id = $1", id); err != nil {
		return sdk.WorkflowNotification{}, sdk.WrapError(err, "loadNotification> Unable to load notification %d sources", id)
	}
	n := sdk.WorkflowNotification(dbnotif)

	for _, id := range n.SourceNodeIDs {
		n.SourceNodeRefs = append(n.SourceNodeRefs, fmt.Sprintf("%d", id))
	}

	return n, nil
}

func insertNotification(db gorp.SqlExecutor, store cache.Store, w *sdk.Workflow, n *sdk.WorkflowNotification, nodes []sdk.WorkflowNode, u *sdk.User) error {
	n.WorkflowID = w.ID
	n.ID = 0
	n.SourceNodeIDs = nil
	dbNotif := Notification(*n)

	//Check references to sources
	if len(n.SourceNodeRefs) == 0 {
		return sdk.WrapError(sdk.ErrWorkflowNodeRef, "insertNotification> No notification references")
	}

	for _, s := range n.SourceNodeRefs {
		//Search references
		var foundRef = findNodeByRef(s, nodes)
		if foundRef == nil || foundRef.ID == 0 {
			return sdk.WrapError(sdk.ErrWorkflowNotificationNodeRef, "insertNotification> Invalid notification references %s", s)
		}
		n.SourceNodeIDs = append(n.SourceNodeIDs, foundRef.ID)
	}

	//Insert the notification
	if err := db.Insert(&dbNotif); err != nil {
		return sdk.WrapError(err, "insertNotification> Unable to insert workflow notification")
	}
	n.ID = dbNotif.ID

	//Insert associations with sources
	query := "insert into workflow_notification_source(workflow_node_id, workflow_notification_id) values ($1, $2)"
	for _, source := range n.SourceNodeIDs {
		if _, err := db.Exec(query, source, n.ID); err != nil {
			return sdk.WrapError(err, "insertNotification> Unable to insert associations between node %d and notification %d", source, n.ID)
		}
	}

	return nil
}

// PostInsert is a db hook
func (no *Notification) PostInsert(db gorp.SqlExecutor) error {
	b, err := json.Marshal(no.Settings)
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
	var res = struct {
		Notification string `db:"settings"`
	}{}

	if err := db.SelectOne(&res, "SELECT settings FROM workflow_notification WHERE id = $1", no.ID); err != nil {
		return sdk.WrapError(err, "PostGet> Unable to load marshalled workflow notification")
	}

	var errN error
	no.Settings, errN = sdk.ParseWorkflowUserNotificationSettings(no.Type, []byte(res.Notification))
	return sdk.WrapError(errN, "Notification.PostGet > Cannot parse user notification")
}
