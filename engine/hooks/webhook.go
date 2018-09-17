package hooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"

	dump "github.com/fsamin/go-dump"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) doWebHookExecution(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing webhook %s %s", t.UUID, t.Type)

	if t.Type == TypeRepoManagerWebHook {
		return executeRepositoryWebHook(t)
	}
	return executeWebHook(t)
}

func getRepositoryHeader(whe *sdk.WebHookExecution) string {
	if v, ok := whe.RequestHeader[GithubHeader]; ok && v[0] == "push" {
		return GithubHeader
	} else if v, ok := whe.RequestHeader[GitlabHeader]; ok && v[0] == "Push Hook" {
		return GitlabHeader
	} else if v, ok := whe.RequestHeader[BitbucketHeader]; ok && v[0] == "repo:refs_changed" {
		return BitbucketHeader
	}
	return ""
}

func executeRepositoryWebHook(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: t.UUID,
	}

	payload := make(map[string]interface{})
	switch getRepositoryHeader(t.WebHook) {
	case GithubHeader:
		var pushEvent GithubPushEvent
		if err := json.Unmarshal(t.WebHook.RequestBody, &pushEvent); err != nil {
			return nil, sdk.WrapError(err, "Hook> webhookHandler> unable ro read github request: %s", string(t.WebHook.RequestBody))
		}
		if pushEvent.Deleted {
			return nil, nil
		}
		payload["git.author"] = pushEvent.HeadCommit.Author.Username
		payload["git.author.email"] = pushEvent.HeadCommit.Author.Email

		if !strings.HasPrefix(pushEvent.Ref, "refs/tags/") {
			payload["git.branch"] = strings.TrimPrefix(pushEvent.Ref, "refs/heads/")
		} else {
			payload["git.tag"] = strings.TrimPrefix(pushEvent.Ref, "refs/tags/")
		}
		payload["git.hash.before"] = pushEvent.Before
		payload["git.hash"] = pushEvent.After
		payload["git.repository"] = pushEvent.Repository.FullName
		payload["cds.triggered_by.username"] = pushEvent.HeadCommit.Author.Username
		payload["cds.triggered_by.fullname"] = pushEvent.HeadCommit.Author.Name
		payload["cds.triggered_by.email"] = pushEvent.HeadCommit.Author.Email

		if len(pushEvent.Commits) > 0 {
			payload["git.message"] = pushEvent.Commits[0].Message
		}
	case GitlabHeader:
		var pushEvent GitlabPushEvent
		if err := json.Unmarshal(t.WebHook.RequestBody, &pushEvent); err != nil {
			return nil, sdk.WrapError(err, "Hook> webhookHandler> unable ro read gitlab request: %s", string(t.WebHook.RequestBody))
		}
		// Branch deletion ( gitlab return 0000000000000000000000000000000000000000 as git hash)
		if pushEvent.After == "0000000000000000000000000000000000000000" {
			return nil, nil
		}
		payload["git.author"] = pushEvent.UserUsername
		payload["git.author.email"] = pushEvent.UserEmail
		if !strings.HasPrefix(pushEvent.Ref, "refs/tags/") {
			payload["git.branch"] = strings.TrimPrefix(pushEvent.Ref, "refs/heads/")
		} else {
			payload["git.tag"] = strings.TrimPrefix(pushEvent.Ref, "refs/tags/")
		}
		payload["git.hash.before"] = pushEvent.Before
		payload["git.hash"] = pushEvent.After
		payload["git.repository"] = pushEvent.Project.PathWithNamespace

		payload["cds.triggered_by.username"] = pushEvent.UserUsername
		payload["cds.triggered_by.fullname"] = pushEvent.UserName
		payload["cds.triggered_by.email"] = pushEvent.UserEmail

		if len(pushEvent.Commits) > 0 {
			payload["git.message"] = pushEvent.Commits[0].Message
		}
	case BitbucketHeader:
		var pushEvent BitbucketPushEvent
		if err := json.Unmarshal(t.WebHook.RequestBody, &pushEvent); err != nil {
			return nil, sdk.WrapError(err, "Hook> webhookHandler> unable ro read bitbucket request: %s", string(t.WebHook.RequestBody))
		}
		payload["git.author"] = pushEvent.Actor.Name
		payload["git.author.email"] = pushEvent.Actor.EmailAddress

		if len(pushEvent.Changes) == 0 || pushEvent.Changes[0].Type == "DELETE" {
			return nil, nil
		}

		if !strings.HasPrefix(pushEvent.Changes[0].RefID, "refs/tags/") {
			payload["git.branch"] = strings.TrimPrefix(pushEvent.Changes[0].RefID, "refs/heads/")
		} else {
			payload["git.tag"] = strings.TrimPrefix(pushEvent.Changes[0].RefID, "refs/tags/")
		}
		payload["git.hash.before"] = pushEvent.Changes[0].FromHash
		payload["git.hash"] = pushEvent.Changes[0].ToHash
		payload["git.repository"] = fmt.Sprintf("%s/%s", pushEvent.Repository.Project.Key, pushEvent.Repository.Slug)

		payload["cds.triggered_by.username"] = pushEvent.Actor.Name
		payload["cds.triggered_by.fullname"] = pushEvent.Actor.DisplayName
		payload["cds.triggered_by.email"] = pushEvent.Actor.EmailAddress
	default:
		log.Warning("executeRepositoryWebHook> Repository manager not found. Cannot read %s", string(t.WebHook.RequestBody))
		return nil, fmt.Errorf("Repository manager not found. Cannot read request body")
	}

	d := dump.NewDefaultEncoder(&bytes.Buffer{})
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
	return &h, nil
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
		return nil, sdk.WrapError(err, "Hooks> Unable to parse query url %s", t.WebHook.RequestURL)
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
		switch {
		case ct == "application/x-www-form-urlencoded":
			formValues, err := url.ParseQuery(string(t.WebHook.RequestBody))
			if err == nil {
				return nil, sdk.WrapError(err, "Hooks> Unable webhookto parse body %s", t.WebHook.RequestBody)
			}
			copyValues(values, formValues)
		case ct == "application/json":
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
			e := dump.NewDefaultEncoder(new(bytes.Buffer))
			e.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
			e.ExtraFields.DetailedMap = false
			e.ExtraFields.DetailedStruct = false
			e.ExtraFields.Len = false
			e.ExtraFields.Type = false
			m, err := e.ToStringMap(bodyJSON)
			if err != nil {
				return nil, sdk.WrapError(err, "Hooks> Unable to dump body %s", t.WebHook.RequestBody)
			}

			//Add the map content to values
			for k, v := range m {
				values.Add(k, v)
			}
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

func copyValues(dst, src url.Values) {
	for k, vs := range src {
		for _, value := range vs {
			dst.Add(k, value)
		}
	}
}
