package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
	"github.com/ovh/cds/sdk/log"
)

var (
	audits = map[string]sdk.Audit{
		fmt.Sprintf("%T", sdk.EventWorkflowAdd{}):              addWorkflowAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowUpdate{}):           updateWorkflowAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowDelete{}):           deleteWorkflowAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowPermissionAdd{}):    addWorkflowPermissionAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowPermissionUpdate{}): updateWorkflowPermissionAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowPermissionDelete{}): deleteWorkflowPermissionAudit{},
	}
)

// ComputeAudit Compute audit on workflow
func ComputeAudit(c context.Context, DBFunc func() *gorp.DbMap) {
	chanEvent := make(chan sdk.Event)
	event.Subscribe(chanEvent)
	deleteTicker := time.NewTicker(15 * time.Minute)
	defer deleteTicker.Stop()

	db := DBFunc()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("ComputeWorkflowAudit> Exiting: %v", c.Err())
				return
			}
		case <-deleteTicker.C:
			if err := purgeAudits(DBFunc()); err != nil {
				log.Error("ComputeWorkflowAudit> Purge error: %v", c)
			}
		case e := <-chanEvent:
			if !strings.HasPrefix(e.EventType, "sdk.EventWorkflow") {
				continue
			}

			if audit, ok := audits[e.EventType]; ok {
				if err := audit.Compute(db, e); err != nil {
					log.Warning("ComputeAudit> Unable to compute audit on event %s: %v", e.EventType, err)
				}
			}
		}
	}
}

type addWorkflowAudit struct{}

func (a addWorkflowAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowAdd
	if err := mapstructure.Decode(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	buffer := bytes.NewBufferString("")
	if _, err := exportWorkflow(wEvent.Workflow, exportentities.FormatYAML, buffer); err != nil {
		return sdk.WrapError(err, "Unable to export workflow")
	}

	return InsertAudit(db, &sdk.AuditWorkflow{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		WorkflowID: wEvent.Workflow.ID,
		ProjectKey: e.ProjectKey,
		DataType:   "yaml",
		DataAfter:  buffer.String(),
	})
}

type updateWorkflowAudit struct{}

func (u updateWorkflowAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowUpdate
	if err := mapstructure.Decode(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	oldWorkflowBuffer := bytes.NewBufferString("")
	if _, err := exportWorkflow(wEvent.OldWorkflow, exportentities.FormatYAML, oldWorkflowBuffer); err != nil {
		return sdk.WrapError(err, "Unable to export workflow")
	}

	newWorkflowBuffer := bytes.NewBufferString("")
	if _, err := exportWorkflow(wEvent.NewWorkflow, exportentities.FormatYAML, newWorkflowBuffer); err != nil {
		return sdk.WrapError(err, "Unable to export workflow")
	}

	return InsertAudit(db, &sdk.AuditWorkflow{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		WorkflowID: wEvent.NewWorkflow.ID,
		ProjectKey: e.ProjectKey,
		DataType:   "yaml",
		DataAfter:  newWorkflowBuffer.String(),
		DataBefore: oldWorkflowBuffer.String(),
	})
}

type deleteWorkflowAudit struct{}

func (d deleteWorkflowAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowDelete
	if err := mapstructure.Decode(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	oldWorkflowBuffer := bytes.NewBufferString("")
	if _, err := exportWorkflow(wEvent.Workflow, exportentities.FormatYAML, oldWorkflowBuffer); err != nil {
		return sdk.WrapError(err, "Unable to export workflow")
	}

	return InsertAudit(db, &sdk.AuditWorkflow{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		WorkflowID: wEvent.Workflow.ID,
		ProjectKey: e.ProjectKey,
		DataType:   "yaml",
		DataBefore: oldWorkflowBuffer.String(),
	})
}

type addWorkflowPermissionAudit struct{}

func (a addWorkflowPermissionAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowPermissionAdd
	if err := mapstructure.Decode(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	b, err := json.MarshalIndent(wEvent.Permission, "", "  ")
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal permission")
	}

	return InsertAudit(db, &sdk.AuditWorkflow{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		WorkflowID: wEvent.WorkflowID,
		ProjectKey: e.ProjectKey,
		DataType:   "json",
		DataAfter:  string(b),
	})
}

type updateWorkflowPermissionAudit struct{}

func (u updateWorkflowPermissionAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowPermissionUpdate
	if err := mapstructure.Decode(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	oldPerm, err := json.MarshalIndent(wEvent.OldPermission, "", "  ")
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal old permission")
	}

	newPerm, err := json.MarshalIndent(wEvent.NewPermission, "", "  ")
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal new permission")
	}

	return InsertAudit(db, &sdk.AuditWorkflow{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		WorkflowID: wEvent.WorkflowID,
		ProjectKey: e.ProjectKey,
		DataType:   "json",
		DataBefore: string(oldPerm),
		DataAfter:  string(newPerm),
	})
}

type deleteWorkflowPermissionAudit struct{}

func (a deleteWorkflowPermissionAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowPermissionDelete
	if err := mapstructure.Decode(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to decode payload")
	}

	b, err := json.MarshalIndent(wEvent.Permission, "", " ")
	if err != nil {
		return sdk.WrapError(err, "Unable to marshal permission")
	}

	return InsertAudit(db, &sdk.AuditWorkflow{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		ProjectKey: e.ProjectKey,
		WorkflowID: wEvent.WorkflowID,
		DataType:   "json",
		DataBefore: string(b),
	})
}

const keepAudits = 50

func purgeAudits(db gorp.SqlExecutor) error {
	var nbAuditsPerWorkflowID = []struct {
		WorkflowID int64 `db:"workflow_id"`
		NbAudits   int64 `db:"nb_audits"`
	}{}

	query := `select workflow_id, count(id) "nb_audits" from workflow_audit group by workflow_id having count(id)  > $1`
	if _, err := db.Select(&nbAuditsPerWorkflowID, query, keepAudits); err != nil {
		return sdk.WithStack(err)
	}

	for _, r := range nbAuditsPerWorkflowID {
		log.Debug("purgeAudits> deleting audits for workflow %d (%d audits)", r.WorkflowID, r.NbAudits)
		var ids []int64
		query = `select id from workflow_audit where workflow_id = $1 order by created asc offset $2`
		if _, err := db.Select(&ids, query, r.WorkflowID, keepAudits); err != nil {
			return sdk.WithStack(err)
		}
		for _, id := range ids {
			if err := deleteAudit(db, id); err != nil {
				log.Error("purgeAudits> unable to delete audit %d: %v", id, err)
			}
		}
	}

	return nil
}

func deleteAudit(db gorp.SqlExecutor, id int64) error {
	_, err := db.Exec(`delete from workflow_audit where id = $1`, id)
	return sdk.WithStack(err)
}
