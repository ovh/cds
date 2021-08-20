package hooks

import (
	"context"

	dump "github.com/fsamin/go-dump"
	"github.com/rockbears/log"

	"github.com/ovh/cds/sdk"
)

func (s *Service) doScheduledTaskExecution(ctx context.Context, t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug(ctx, "Hooks> Processing scheduled task %s", t.UUID)

	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: t.UUID,
	}

	//Prepare the payload
	//Anything can be pushed in the configuration, just avoid sending

	payloadValues := map[string]string{}
	if payload, ok := t.Config[sdk.Payload]; ok && payload.Value != "{}" {
		var payloadInt interface{}
		if err := sdk.JSONUnmarshal([]byte(payload.Value), &payloadInt); err == nil {
			e := dump.NewDefaultEncoder()
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false

			m1, errm1 := e.ToStringMap(payloadInt)
			if errm1 != nil {
				log.Error(ctx, "Hooks> doScheduledTaskExecution> Cannot convert payload to map %s", errm1)
			} else {
				payloadValues = m1
			}
			payloadValues["payload"] = payload.Value
		} else {
			log.Error(ctx, "Hooks> doScheduledTaskExecution> Cannot unmarshall payload %s", err)
		}
	}
	for k, v := range t.Config {
		switch k {
		case sdk.HookConfigProject, sdk.HookConfigWorkflow, sdk.SchedulerModelCron, sdk.SchedulerModelTimezone, sdk.Payload:
		default:
			payloadValues[k] = v.Value
		}
	}
	payloadValues["cds.triggered_by.username"] = sdk.SchedulerUsername
	payloadValues["cds.triggered_by.fullname"] = sdk.SchedulerFullname
	h.Payload = payloadValues

	return &h, nil
}
