package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"

	dump "github.com/fsamin/go-dump"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) doWebHookExecution(ctx context.Context, e *sdk.TaskExecution) ([]sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing webhook %s %s", e.UUID, e.Type)

	if e.Type == TypeRepoManagerWebHook {
		return s.executeRepositoryWebHook(ctx, e)
	}
	event, err := executeWebHook(e)
	if err != nil {
		return nil, err
	}
	return []sdk.WorkflowNodeRunHookEvent{*event}, nil
}

func getRepositoryHeader(t *sdk.TaskExecution, events []string) string {
	if v, ok := t.WebHook.RequestHeader[GithubHeader]; ok && ((len(events) == 0 && v[0] == "push") || sdk.IsInArray(v[0], events)) {
		return GithubHeader
	} else if v, ok := t.WebHook.RequestHeader[GitlabHeader]; ok && ((len(events) == 0 && (v[0] == string(gitlab.EventTypePush) || v[0] == string(gitlab.EventTypeTagPush))) || sdk.IsInArray(v[0], events)) {
		return GitlabHeader
	} else if v, ok := t.WebHook.RequestHeader[BitbucketHeader]; ok {
		if sdk.IsInArray(v[0], events) || (len(events) == 0 && (v[0] == "repo:refs_changed" || v[0] == "repo:push")) {
			var request sdk.BitbucketServerWebhookEvent
			if err := json.Unmarshal(t.WebHook.RequestBody, &request); err == nil && request.EventKey != "" {
				return BitbucketHeader
			} else {
				// We return a fake header to make a difference between server and cloud version
				return BitbucketCloudHeader
			}
		}
	}
	return ""
}

func (s *Service) executeRepositoryWebHook(ctx context.Context, t *sdk.TaskExecution) ([]sdk.WorkflowNodeRunHookEvent, error) {
	// Prepare a struct to send to CDS API
	payloads := []map[string]interface{}{}

	var events []string
	if _, ok := t.Config[sdk.HookConfigEventFilter]; ok && t.Config[sdk.HookConfigEventFilter].Value != "" {
		events = strings.Split(t.Config[sdk.HookConfigEventFilter].Value, ";")
	}

	switch getRepositoryHeader(t, events) {
	case GithubHeader:
		headerValue := t.WebHook.RequestHeader[GithubHeader][0]
		payload, err := s.generatePayloadFromGithubRequest(ctx, t, headerValue)
		if err != nil {
			return nil, err
		}
		if payload != nil {
			payloads = append(payloads, payload)
		}
	case GitlabHeader:
		headerValue := t.WebHook.RequestHeader[GitlabHeader][0]
		payload, err := s.generatePayloadFromGitlabRequest(ctx, t, headerValue)
		if err != nil {
			return nil, err
		}
		if payload != nil {
			payloads = append(payloads, payload)
		}
	case BitbucketHeader:
		headerValue := t.WebHook.RequestHeader[BitbucketHeader][0]
		var errG error
		payloads, errG = s.generatePayloadFromBitbucketServerRequest(ctx, t, headerValue)
		if errG != nil {
			return nil, errG
		}
	case BitbucketCloudHeader:
		headerValue := t.WebHook.RequestHeader[BitbucketHeader][0]
		var errG error
		payloads, errG = s.generatePayloadFromBitbucketCloudRequest(ctx, t, headerValue)
		if errG != nil {
			return nil, errG
		}
	default:
		log.Warning(ctx, "executeRepositoryWebHook> Repository manager not found. Cannot read %s", string(t.WebHook.RequestBody))
		return nil, fmt.Errorf("Repository manager not found. Cannot read request body")
	}

	hs := make([]sdk.WorkflowNodeRunHookEvent, 0, len(payloads))
	for _, payload := range payloads {
		h := sdk.WorkflowNodeRunHookEvent{
			WorkflowNodeHookUUID: t.UUID,
		}
		d := dump.NewDefaultEncoder()
		d.ExtraFields.Type = false
		d.ExtraFields.Len = false
		d.ExtraFields.DetailedMap = false
		d.ExtraFields.DetailedStruct = false
		d.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
		payloadValues, errDump := d.ToStringMap(payload)
		if errDump != nil {
			return nil, sdk.WrapError(errDump, "executeRepositoryWebHook> Cannot dump payload %+v ", payload)
		}
		h.Payload = payloadValues
		hs = append(hs, h)
	}

	return hs, nil
}

func executeWebHook(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: t.UUID,
		Payload:              map[string]string{},
	}

	// Compute the payload, from the header, the body and the url
	// For all requests, parse the raw query from the URL
	values, err := url.ParseQuery(t.WebHook.RequestURL)
	if err != nil {
		return nil, sdk.WrapError(err, "Unable to parse query url %s", t.WebHook.RequestURL)
	}

	// For POST, PUT, and PATCH requests, it also parses the request body as a form
	confMethod := t.Config[sdk.WebHookModelConfigMethod]
	if confMethod.Value == "POST" || confMethod.Value == "PUT" || confMethod.Value == "PATCH" {
		//Depending on the content type, we should not read the body the same way
		header := http.Header(t.WebHook.RequestHeader)
		ct := header.Get("Content-Type")
		// RFC 2616, section 7.2.1 - empty type
		//   SHOULD be treated as application/octet-stream
		if ct == "" {
			ct = "application/octet-stream"
		}
		//Parse the content type
		ct, _, _ = mime.ParseMediaType(ct)
		switch ct {
		case "application/x-www-form-urlencoded":
			formValues, err := url.ParseQuery(string(t.WebHook.RequestBody))
			if err != nil {
				return nil, sdk.WrapError(err, "Unable webhook to parse body %s", t.WebHook.RequestBody)
			}
			copyValues(values, formValues)
			h.Payload["payload"] = string(t.WebHook.RequestBody)
		case "application/json":
			var bodyJSON interface{}

			//Try to parse the body as an array
			bodyJSONArray := []interface{}{}
			if err := json.Unmarshal(t.WebHook.RequestBody, &bodyJSONArray); err != nil {

				//Try to parse the body as a map
				bodyJSONMap := map[string]interface{}{}
				if err2 := json.Unmarshal(t.WebHook.RequestBody, &bodyJSONMap); err2 == nil {
					bodyJSON = bodyJSONMap
				}
			} else {
				bodyJSON = bodyJSONArray
			}

			//Go Dump
			e := dump.NewDefaultEncoder()
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false
			m, err := e.ToStringMap(bodyJSON)
			if err != nil {
				return nil, sdk.WrapError(err, "Unable to dump body %s", t.WebHook.RequestBody)
			}

			//Add the map content to values
			for k, v := range m {
				values.Add(k, v)
			}
			h.Payload["payload"] = string(t.WebHook.RequestBody)
		default:
			h.Payload["payload"] = t.WebHook.RequestURL
		}
	} else {
		if _, ok := h.Payload["payload"]; !ok {
			h.Payload["payload"] = t.WebHook.RequestURL
		}
	}

	//Prepare the payload
	for k, v := range t.Config {
		switch k {
		case sdk.HookConfigProject, sdk.HookConfigWorkflow, sdk.WebHookModelConfigMethod:
		default:
			h.Payload[k] = v.Value
		}
	}

	h.Payload["cds.triggered_by.username"] = "cds.webhook"
	h.Payload["cds.triggered_by.fullname"] = "CDS Webhook"

	//try to find some specific values
	for k := range values {
		h.Payload[k] = values.Get(k)
	}
	return &h, nil
}

func (s *Service) enqueueBranchDeletion(projectKey, workflowName, branch string) error {
	config := sdk.WorkflowNodeHookConfig{
		"project": sdk.WorkflowNodeHookConfigValue{
			Configurable: false,
			Type:         sdk.HookConfigTypeProject,
			Value:        projectKey,
		},
		"workflow": sdk.WorkflowNodeHookConfigValue{
			Configurable: false,
			Type:         sdk.HookConfigTypeWorkflow,
			Value:        workflowName,
		},
		"branch": sdk.WorkflowNodeHookConfigValue{
			Configurable: false,
			Type:         sdk.HookConfigTypeString,
			Value:        branch,
		},
	}
	task := sdk.Task{
		Config: config,
		Type:   TypeBranchDeletion,
		UUID:   branch + "-" + sdk.UUID(),
	}

	_, err := s.startTask(context.Background(), &task)

	return sdk.WrapError(err, "cannot start task")
}

func copyValues(dst, src url.Values) {
	for k, vs := range src {
		for _, value := range vs {
			dst.Add(k, value)
		}
	}
}
