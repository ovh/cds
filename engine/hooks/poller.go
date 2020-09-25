package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	dump "github.com/fsamin/go-dump"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func fillPayload(ctx context.Context, pushEvent sdk.VCSPushEvent) map[string]string {
	payload := make(map[string]string)
	payload["git.author"] = pushEvent.Commit.Author.Name
	payload["git.author.email"] = pushEvent.Commit.Author.Email
	payload["git.branch"] = strings.TrimPrefix(strings.TrimPrefix(pushEvent.Branch.DisplayID, "refs/heads/"), "refs/tags/")
	payload["git.hash"] = pushEvent.Commit.Hash
	payload["git.hash.short"] = sdk.StringFirstN(pushEvent.Commit.Hash, 7)
	payload["git.repository"] = pushEvent.Repo
	payload["cds.triggered_by.username"] = pushEvent.Commit.Author.DisplayName
	payload["cds.triggered_by.fullname"] = pushEvent.Commit.Author.Name
	payload["cds.triggered_by.email"] = pushEvent.Commit.Author.Email
	payload["git.message"] = pushEvent.Commit.Message

	payloadStr, err := json.Marshal(pushEvent)
	if err != nil {
		log.Error(ctx, "Unable to marshal payload: %v", err)
	}
	payload["payload"] = string(payloadStr)

	if strings.HasPrefix(pushEvent.Branch.DisplayID, "refs/tags/") {
		payload["git.tag"] = strings.TrimPrefix(pushEvent.Branch.DisplayID, "refs/tags/")
	}

	return payload
}

func (s *Service) doPollerTaskExecution(ctx context.Context, task *sdk.Task, taskExec *sdk.TaskExecution) ([]sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing polling task %s:%d", taskExec.UUID, taskExec.Timestamp)

	tExecs, errF := s.Dao.FindAllTaskExecutions(ctx, task)
	if errF != nil {
		return nil, errF
	}

	var maxTs int64
	// get max timestamp for previous tasks execution
	for _, tExec := range tExecs {
		if tExec.Status == TaskExecutionDone && maxTs < tExec.Timestamp {
			maxTs = tExec.Timestamp
		}
	}
	workflowID, errP := strconv.ParseInt(taskExec.Config[sdk.HookConfigWorkflowID].Value, 10, 64)
	if errP != nil {
		return nil, sdk.WrapError(errP, "Hooks> doPollerTaskExecution> Cannot convert workflow id %s", taskExec.Config[sdk.HookConfigWorkflowID].Value)
	}
	events, interval, err := s.Client.PollVCSEvents(taskExec.UUID, workflowID, taskExec.Config["vcsServer"].Value, maxTs)
	if err != nil {
		return nil, sdk.WrapError(err, "Cannot poll vcs events for workflow %s with vcsserver %s", taskExec.Config[sdk.HookConfigWorkflow].Value, taskExec.Config["vcsServer"].Value)
	}

	//Prepare the payload
	//Anything can be pushed in the configuration, just avoid sending
	payloadValues := map[string]string{}
	if payload, ok := task.Config["payload"]; ok && payload.Value != "{}" {
		var payloadInt interface{}
		if err := json.Unmarshal([]byte(payload.Value), &payloadInt); err == nil {
			e := dump.NewDefaultEncoder()
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false

			m1, errm1 := e.ToStringMap(payloadInt)
			if errm1 != nil {
				log.Error(ctx, "Hooks> doPollerTaskExecution> Cannot convert payload to map %s", errm1)
			} else {
				payloadValues = m1
			}
		} else {
			log.Error(ctx, "Hooks> doPollerTaskExecution> Cannot unmarshall payload %s", err)
		}
		payloadValues["payload"] = string(payload.Value)
	}

	var hookEvents []sdk.WorkflowNodeRunHookEvent
	if len(events.PushEvents) > 0 || len(events.PullRequestEvents) > 0 {
		i := 0
		hookEvents = make([]sdk.WorkflowNodeRunHookEvent, len(events.PushEvents)+len(events.PullRequestEvents))
		for _, pushEvent := range events.PushEvents {
			payload := fillPayload(ctx, pushEvent)
			hookEvents[i] = sdk.WorkflowNodeRunHookEvent{
				WorkflowNodeHookUUID: task.UUID,
				Payload:              sdk.ParametersMapMerge(payloadValues, payload),
			}
			i++
		}

		for _, pullRequestEvent := range events.PullRequestEvents {
			payload := fillPayload(ctx, pullRequestEvent.Head)
			hookEvents[i] = sdk.WorkflowNodeRunHookEvent{
				WorkflowNodeHookUUID: task.UUID,
				Payload:              sdk.ParametersMapMerge(payloadValues, payload),
			}
			i++
		}
	}

	nextExec := fmt.Sprint(time.Now().Add(interval).Unix())
	taskExec.Config["next_execution"] = sdk.WorkflowNodeHookConfigValue{
		Configurable: false,
		Value:        nextExec,
	}
	taskExec.ScheduledTask.DateScheduledExecution = nextExec

	return hookEvents, nil
}
