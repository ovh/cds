package action

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/engine/api/event"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

var (
	audits = map[string]sdk.Audit{
		fmt.Sprintf("%T", sdk.EventActionAdd{}):    addActionAudit{},
		fmt.Sprintf("%T", sdk.EventActionUpdate{}): updateActionAudit{},
	}
)

// ComputeAudit compute audit on action.
func ComputeAudit(c context.Context, DBFunc func() *gorp.DbMap) {
	chanEvent := make(chan sdk.Event)
	event.Subscribe(chanEvent)

	db := DBFunc()
	for {
		select {
		case <-c.Done():
			if c.Err() != nil {
				log.Error("%v", sdk.WithStack(c.Err()))
				return
			}
		case e := <-chanEvent:
			if !strings.HasPrefix(e.EventType, "sdk.EventAction") {
				continue
			}

			if audit, ok := audits[e.EventType]; ok {
				if err := audit.Compute(db, e); err != nil {
					log.Warning("%v", sdk.WrapError(err, "unable to compute audit on event %s", e.EventType))
				}
			}
		}
	}
}

type addActionAudit struct{}

func (a addActionAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var aEvent sdk.EventActionAdd
	if err := mapstructure.Decode(e.Payload, &aEvent); err != nil {
		return sdk.WrapError(err, "unable to decode payload")
	}

	b, err := json.Marshal(aEvent.Action)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal action")
	}

	return insertAudit(db, &sdk.AuditAction{
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

func (a updateActionAudit) Compute(db gorp.SqlExecutor, e sdk.Event) error {
	var aEvent sdk.EventActionUpdate
	if err := mapstructure.Decode(e.Payload, &aEvent); err != nil {
		return sdk.WrapError(err, "unable to decode payload")
	}

	before, err := json.Marshal(aEvent.OldAction)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal action")
	}

	after, err := json.Marshal(aEvent.NewAction)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal action")
	}

	return insertAudit(db, &sdk.AuditAction{
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
