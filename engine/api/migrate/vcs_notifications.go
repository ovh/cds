package migrate

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/engine/api/workflow"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func AddDefaultVCSNotifications(ctx context.Context, store cache.Store, DBFunc func() *gorp.DbMap) error {
	db := DBFunc()

	res := []struct {
		WorkflowID int64 `db:"workflow_id"`
		NodeID     int64 `db:"id"`
	}{}

	query := `SELECT w_node.workflow_id, w_node.id
	FROM w_node
		JOIN w_node_context ON w_node_context.node_id = w_node.id
		JOIN application ON application.id = w_node_context.application_id
	WHERE application.repo_fullname <> ''`

	if _, err := db.Select(&res, query); err != nil {
		return sdk.WrapError(err, "cannot get workflow id and node id to update")
	}

	// switch to a hashmap of workflow id with key
	nodesByWorkflowID := map[int64][]int64{}
	for _, resp := range res {
		nodesByWorkflowID[resp.WorkflowID] = append(nodesByWorkflowID[resp.WorkflowID], resp.NodeID)
	}

	for workflowID, nodes := range nodesByWorkflowID {

		count, err := db.SelectInt("SELECT COUNT(id) FROM workflow_notification WHERE workflow_id = $1 and type = $2", workflowID, sdk.VCSUserNotification)
		if err != nil {
			log.Error("migrate.AddDefaultVCSNotifications> cannot count workflow_notification for workflow id %d", workflowID)
			continue
		}
		if count != 0 {
			continue
		}

		notif := sdk.WorkflowNotification{
			Settings: sdk.UserNotificationSettings{
				Template: &sdk.UserNotificationTemplate{
					Body: sdk.DefaultWorkflowNodeRunReport,
				},
			},
			WorkflowID: workflowID,
			Type:       sdk.VCSUserNotification,
			NodeIDs:    nodes,
		}
		dbNotif := workflow.Notification(notif)

		tx, err := db.Begin()
		if err != nil {
			log.Error("migrate.AddDefaultVCSNotifications> cannot begin transaction for workflow id %d : %v", workflowID, err)
			continue
		}

		//Insert the notification
		if err := tx.Insert(&dbNotif); err != nil {
			_ = tx.Rollback()
			log.Error("migrate.AddDefaultVCSNotifications> Unable to insert workflow notification : %v", err)
			continue
		}
		notif.ID = dbNotif.ID

		//Insert associations with sources
		query := "INSERT INTO workflow_notification_source(workflow_notification_id, node_id) VALUES ($1, $2) ON CONFLICT DO NOTHING"
		for i := range notif.NodeIDs {
			if _, err := tx.Exec(query, notif.ID, notif.NodeIDs[i]); err != nil {
				_ = tx.Rollback()
				log.Error("migrate.AddDefaultVCSNotifications> Unable to insert associations between node %d and notification %d : %v", notif.NodeIDs[i], notif.ID, err)
				continue
			}
		}

		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			log.Error("migrate.AddDefaultVCSNotifications> cannot commit transaction for workflow id %d : %v", workflowID, err)
		}
	}

	return nil
}
