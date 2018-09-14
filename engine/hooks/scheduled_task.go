package hooks

import (
	"bytes"
	"encoding/json"

	dump "github.com/fsamin/go-dump"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) doScheduledTaskExecution(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing scheduled task %s", t.UUID)

	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: t.UUID,
	}

	//Prepare the payload
	//Anything can be pushed in the configuration, just avoid sending
	payloadValues := map[string]string{}
	if payload, ok := t.Config[sdk.Payload]; ok && payload.Value != "{}" {
		var payloadInt interface{}
		if err := json.Unmarshal([]byte(payload.Value), &payloadInt); err == nil {
			e := dump.NewDefaultEncoder(new(bytes.Buffer))
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false

			m1, errm1 := e.ToStringMap(payloadInt)
			if errm1 != nil {
				log.Error("Hooks> doScheduledTaskExecution> Cannot convert payload to map %s", errm1)
			} else {
				payloadValues = m1
			}
		} else {
			log.Error("Hooks> doScheduledTaskExecution> Cannot unmarshall payload %s", err)
		}
	}
	for k, v := range t.Config {
		switch k {
		case sdk.HookConfigProject, sdk.HookConfigWorkflow, sdk.SchedulerModelCron, sdk.SchedulerModelTimezone, sdk.Payload:
		default:
			payloadValues[k] = v.Value
		}
	}
	payloadValues["cds.triggered_by.username"] = "cds.scheduler"
	payloadValues["cds.triggered_by.fullname"] = "CDS Scheduler"
	h.Payload = payloadValues

	return &h, nil
}
