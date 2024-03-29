package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"
	"gopkg.in/yaml.v2"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/exportentities"
)

var (
	Audits = map[string]sdk.Audit{
		fmt.Sprintf("%T", sdk.EventWorkflowAdd{}):              addWorkflowAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowUpdate{}):           updateWorkflowAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowPermissionAdd{}):    addWorkflowPermissionAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowPermissionUpdate{}): updateWorkflowPermissionAudit{},
		fmt.Sprintf("%T", sdk.EventWorkflowPermissionDelete{}): deleteWorkflowPermissionAudit{},
	}
)

type addWorkflowAudit struct{}

func (a addWorkflowAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowAdd
	if err := sdk.JSONUnmarshal(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
	}

	w, err := exportentities.NewWorkflow(ctx, wEvent.Workflow)
	if err != nil {
		return sdk.WrapError(err, "Unable to export workflow")
	}
	buf, err := yaml.Marshal(w)
	if err != nil {
		return sdk.WithStack(err)
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
		DataAfter:  string(buf),
	})
}

type updateWorkflowAudit struct{}

func (u updateWorkflowAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowUpdate
	if err := sdk.JSONUnmarshal(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
	}

	old, err := exportentities.NewWorkflow(ctx, wEvent.OldWorkflow)
	if err != nil {
		return sdk.WrapError(err, "Unable to export workflow")
	}
	oldWorkflowBuffer, err := yaml.Marshal(old)
	if err != nil {
		return sdk.WithStack(err)
	}

	newW, err := exportentities.NewWorkflow(ctx, wEvent.NewWorkflow)
	if err != nil {
		return sdk.WrapError(err, "Unable to export workflow")
	}
	newWorkflowBuffer, err := yaml.Marshal(newW)
	if err != nil {
		return sdk.WithStack(err)
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
		DataAfter:  string(newWorkflowBuffer),
		DataBefore: string(oldWorkflowBuffer),
	})
}

type addWorkflowPermissionAudit struct{}

func (a addWorkflowPermissionAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowPermissionAdd
	if err := sdk.JSONUnmarshal(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
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

func (u updateWorkflowPermissionAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowPermissionUpdate
	if err := sdk.JSONUnmarshal(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
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

func (a deleteWorkflowPermissionAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var wEvent sdk.EventWorkflowPermissionDelete
	if err := sdk.JSONUnmarshal(e.Payload, &wEvent); err != nil {
		return sdk.WrapError(err, "Unable to unmarshal payload")
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

func PurgeAudits(ctx context.Context, db gorp.SqlExecutor) error {
	var nbAuditsPerWorkflowID = []struct {
		WorkflowID int64 `db:"workflow_id"`
		NbAudits   int64 `db:"nb_audits"`
	}{}

	query := `select workflow_id, count(id) "nb_audits" from workflow_audit group by workflow_id having count(id)  > $1`
	if _, err := db.Select(&nbAuditsPerWorkflowID, query, keepAudits); err != nil {
		return sdk.WithStack(err)
	}

	for _, r := range nbAuditsPerWorkflowID {
		log.Debug(ctx, "purgeAudits> deleting audits for workflow %d (%d audits)", r.WorkflowID, r.NbAudits)
		var ids []int64
		query = `select id from workflow_audit where workflow_id = $1 order by created desc offset $2`
		if _, err := db.Select(&ids, query, r.WorkflowID, keepAudits); err != nil {
			return sdk.WithStack(err)
		}
		for _, id := range ids {
			if err := deleteAudit(db, id); err != nil {
				log.Error(ctx, "purgeAudits> unable to delete audit %d: %v", id, err)
			}
		}
	}

	return nil
}

func deleteAudit(db gorp.SqlExecutor, id int64) error {
	_, err := db.Exec(`delete from workflow_audit where id = $1`, id)
	return sdk.WithStack(err)
}
