package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

var (
	audits = map[string]sdk.Audit{
		fmt.Sprintf("%T", sdk.EventActionAdd{}):    addActionAudit{},
		fmt.Sprintf("%T", sdk.EventActionUpdate{}): updateActionAudit{},
	}
)

// ComputeAudit compute audit on action.
func ComputeAudit(ctx context.Context, DBFunc func() *gorp.DbMap, chanEvent chan sdk.Event) {
	db := DBFunc()
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() != nil {
				log.Error(ctx, "%v", sdk.WithStack(ctx.Err()))
				return
			}
		case e := <-chanEvent:
			if !strings.HasPrefix(e.EventType, "sdk.EventAction") {
				continue
			}

			if audit, ok := audits[e.EventType]; ok {
				if err := audit.Compute(ctx, db, e); err != nil {
					log.Warn(ctx, "%v", sdk.WrapError(err, "unable to compute audit on event %s", e.EventType))
				}
			}
		}
	}
}

type addActionAudit struct{}

func (a addActionAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var aEvent sdk.EventActionAdd
	if err := sdk.JSONUnmarshal(e.Payload, &aEvent); err != nil {
		return sdk.WrapError(err, "unable to unmarshal payload")
	}

	b, err := json.Marshal(aEvent.Action)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal action")
	}

	return InsertAudit(db, &sdk.AuditAction{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		ActionID:  aEvent.Action.ID,
		DataType:  "json",
		DataAfter: string(b),
	})
}

type updateActionAudit struct{}

func (a updateActionAudit) Compute(ctx context.Context, db gorp.SqlExecutor, e sdk.Event) error {
	var aEvent sdk.EventActionUpdate
	if err := sdk.JSONUnmarshal(e.Payload, &aEvent); err != nil {
		return sdk.WrapError(err, "unable to unmarshal payload")
	}

	before, err := json.Marshal(aEvent.OldAction)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal action")
	}

	after, err := json.Marshal(aEvent.NewAction)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal action")
	}

	return InsertAudit(db, &sdk.AuditAction{
		AuditCommon: sdk.AuditCommon{
			EventType:   strings.Replace(e.EventType, "sdk.Event", "", -1),
			Created:     e.Timestamp,
			TriggeredBy: e.Username,
		},
		ActionID:   aEvent.NewAction.ID,
		DataType:   "json",
		DataBefore: string(before),
		DataAfter:  string(after),
	})
}
