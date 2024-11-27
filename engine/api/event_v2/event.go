package event_v2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sync"
	"text/template"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/ovh/cds/sdk/interpolate"
	cdslog "github.com/ovh/cds/sdk/log"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/notification_v2"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/engine/api/workflow_v2"
	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/engine/gorpmapper"
)

const (
	eventQueue      = "events_v2"
	EventUIWS       = "event:ui"
	EventHatcheryWS = "event:run:job"
)

var httpClient = cdsclient.NewHTTPClient(10*time.Second, false)

// Enqueue event into cache
func publish(ctx context.Context, store cache.Store, event interface{}) {
	if err := store.Enqueue(eventQueue, event); err != nil {
		log.Error(ctx, "EventV2.publish: %s", err)
		return
	}
}

// Dequeue runs in a goroutine and dequeue event from cache
func Dequeue(ctx context.Context, db *gorp.DbMap, store cache.Store, goroutines *sdk.GoRoutines, cdsUIURL string) {
	for {
		if err := ctx.Err(); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "EventV2.DequeueEvent> Exiting: %v", err)
			return
		}
		var event sdk.FullEventV2
		if err := store.DequeueWithContext(ctx, eventQueue, 50*time.Millisecond, &event); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "EventV2.DequeueEvent> store.DequeueWithContext err: %v", err)
			continue
		}

		wg := sync.WaitGroup{}

		// Push to elasticsearch
		wg.Add(1)
		goroutines.Exec(ctx, "event.pushToElasticSearch", func(ctx context.Context) {
			defer wg.Done()
			if err := pushToElasticSearch(ctx, db, event); err != nil {
				log.Error(ctx, "EventV2.pushToElasticSearch: %v", err)
			}
		})

		// Create audit
		wg.Add(1)
		goroutines.Exec(ctx, "event.audit", func(_ context.Context) {
			defer wg.Done()
			// TODO Audit
		})

		// Push to websockets channels
		wg.Add(1)
		goroutines.Exec(ctx, "event.websockets", func(ctx context.Context) {
			defer wg.Done()
			pushToWebsockets(ctx, store, event)
		})

		// Project notifications
		wg.Add(1)
		goroutines.Exec(ctx, "event.notifications", func(ctx context.Context) {
			defer wg.Done()
			if err := pushNotifications(ctx, db, event); err != nil {
				log.Error(ctx, "EventV2.pushNotifications: %v", err)
			}
		})

		// Workflow notifications
		wg.Add(1)
		goroutines.Exec(ctx, "event.workflow.notifications", func(ctx context.Context) {
			defer wg.Done()
			ctx = context.WithValue(ctx, cdslog.Project, event.ProjectKey)
			ctx = context.WithValue(ctx, cdslog.VCSServer, event.VCSName)
			ctx = context.WithValue(ctx, cdslog.Repository, event.Repository)
			ctx = context.WithValue(ctx, cdslog.Workflow, event.Workflow)
			ctx = context.WithValue(ctx, cdslog.WorkflowRunID, event.WorkflowRunID)
			if err := workflowNotifications(ctx, db, store, event, cdsUIURL); err != nil {
				ctx := log.ContextWithStackTrace(ctx, err)
				log.Error(ctx, "EventV2.workflowNotifications: %v", err)
			}
		})

		wg.Wait()
	}
}

func pushToWebsockets(ctx context.Context, store cache.Store, event sdk.FullEventV2) {
	msg, err := json.Marshal(event)
	if err != nil {
		log.Error(ctx, "EventV2.pushToWebsockets: unable to marshal event: %v", err)
		return
	}
	if err := store.Publish(ctx, EventUIWS, string(msg)); err != nil {
		log.Error(ctx, "EventV2.pushToWebsockets: ui: %v", err)
	}

	if event.Type == sdk.EventRunJobEnqueued {
		if err := store.Publish(ctx, EventHatcheryWS, string(msg)); err != nil {
			log.Error(ctx, "EventV2.pushToWebsockets: hatchery: %v", err)
		}
	}
}

func workflowNotifications(ctx context.Context, db *gorp.DbMap, store cache.Store, event sdk.FullEventV2, cdsUIURL string) error {
	if event.ProjectKey == "" {
		return nil
	}

	if event.Type != sdk.EventRunEnded && event.Type != sdk.EventRunBuilding {
		return nil
	}

	//event.Payload contains a EventWorkflowRunPayload
	var run sdk.EventWorkflowRunPayload
	if err := json.Unmarshal(event.Payload, &run); err != nil {
		return sdk.WrapError(err, "cannot read payload for type %q", event.Type)
	}

	// always send build status on workflow End
	vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, store, event.ProjectKey, event.VCSName)
	if err != nil {
		return sdk.WrapError(err, "can't get AuthorizedClient for %v/%v", event.ProjectKey, event.VCSName)
	}

	title := fmt.Sprintf("%s-%s", event.ProjectKey, run.WorkflowName)
	description := run.WorkflowName + ":" + string(run.Status)
	if run.WorkflowData.Workflow.CommitStatus != nil {
		if run.WorkflowData.Workflow.CommitStatus.Title != "" {
			title = run.WorkflowData.Workflow.CommitStatus.Title
		}
		if run.WorkflowData.Workflow.CommitStatus.Description != "" {
			description = run.WorkflowData.Workflow.CommitStatus.Description
		}
	}

	buildStatus := sdk.VCSBuildStatus{
		Title:              title,
		Description:        description,
		URLCDS:             fmt.Sprintf("%s/project/%s/run/%s", cdsUIURL, event.ProjectKey, event.WorkflowRunID),
		Context:            fmt.Sprintf("%s-%s #%d", event.ProjectKey, run.WorkflowName, run.RunNumber),
		Status:             event.Status,
		RepositoryFullname: event.Repository,
		GitHash:            run.Contexts.Git.Sha,
	}
	if err := vcsClient.SetStatus(ctx, buildStatus); err != nil {
		return sdk.WrapError(err, "can't send the build status for %v/%v", event.ProjectKey, event.VCSName)
	}

	if event.Type != sdk.EventRunEnded {
		return nil
	}

	if run.WorkflowData.Workflow.On == nil {
		return nil
	}

	if run.WorkflowData.Workflow.On.PullRequest != nil {
		if run.WorkflowData.Workflow.On.PullRequest.Comment != "" && run.Contexts.Git.PullRequestID != 0 {

			err := sendVCSPullRequestComment(ctx, db, vcsClient, run, run.WorkflowData.Workflow.On.PullRequest.Comment)
			if err != nil {
				return sdk.WrapError(err, "can't send the pull-request comment for %v/%v", event.ProjectKey, event.VCSName)
			}
		}
	}

	if run.WorkflowData.Workflow.On.PullRequestComment != nil {
		if run.WorkflowData.Workflow.On.PullRequestComment.Comment != "" && run.Contexts.Git.PullRequestID != 0 {
			err := sendVCSPullRequestComment(ctx, db, vcsClient, run, run.WorkflowData.Workflow.On.PullRequestComment.Comment)
			if err != nil {
				return sdk.WrapError(err, "can't send the pull-request comment for %v/%v", event.ProjectKey, event.VCSName)
			}
		}
	}

	return nil
}

func sendVCSPullRequestComment(ctx context.Context, db *gorp.DbMap, vcsClient sdk.VCSAuthorizedClientService, run sdk.EventWorkflowRunPayload, comment string) error {
	pr, err := vcsClient.PullRequest(ctx, run.Contexts.Git.Repository, fmt.Sprintf("%d", run.Contexts.Git.PullRequestID))
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return err
	}

	if pr.Merged || pr.Closed {
		log.Info(ctx, "nothing to do on pr %d", pr, run.Contexts.Git.PullRequestID)
		return nil
	}

	bts, _ := json.Marshal(run.Contexts)
	var runContext map[string]interface{}
	_ = json.Unmarshal(bts, &runContext)

	tmplParams := make(map[string]interface{})
	for k, v := range runContext {
		tmplParams[k] = v
	}
	eventCtx := sdk.EventWorkflowRunPayloadContextEvent{
		Status:   string(run.Status),
		AdminMFA: run.AdminMFA,
	}
	btsEvent, _ := json.Marshal(eventCtx)
	var eventContext map[string]interface{}
	_ = json.Unmarshal(btsEvent, &eventContext)
	tmplParams["event"] = eventContext

	// Templating
	tmpl, err := template.New("workflow_template").Funcs(interpolate.InterpolateHelperFuncs).Delims("[[", "]]").Parse(comment)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		runInfo := sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			IssuedAt:      time.Now(),
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("unable to parse pullrequest comment: %v", err),
		}

		tx, errT := db.Begin()
		if errT != nil {
			log.ErrorWithStackTrace(ctx, errT)
			// Return original error
			return err
		}
		if errT := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); errT != nil {
			log.ErrorWithStackTrace(ctx, errT)
			// Return original error
			return err
		}
		if errT := tx.Commit(); errT != nil {
			log.ErrorWithStackTrace(ctx, errT)
			// Return original error
			return err
		}
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplParams); err != nil {
		log.ErrorWithStackTrace(ctx, err)
		runInfo := sdk.V2WorkflowRunInfo{
			WorkflowRunID: run.ID,
			IssuedAt:      time.Now(),
			Level:         sdk.WorkflowRunInfoLevelError,
			Message:       fmt.Sprintf("unable to execute template on pullrequest comment: %v", err),
		}

		tx, errT := db.Begin()
		if errT != nil {
			log.ErrorWithStackTrace(ctx, errT)
			// Return original error
			return err
		}
		if errT := workflow_v2.InsertRunInfo(ctx, tx, &runInfo); errT != nil {
			log.ErrorWithStackTrace(ctx, errT)
			// Return original error
			return err
		}
		if errT := tx.Commit(); errT != nil {
			log.ErrorWithStackTrace(ctx, errT)
			// Return original error
			return err
		}
		return err
	}

	prComment := sdk.VCSPullRequestCommentRequest{
		Revision: run.Contexts.Git.Ref,
		Message:  buf.String(),
		ID:       int(run.Contexts.Git.PullRequestID),
	}

	log.Info(ctx, "send comment (revision: %v pr: %v) on repo %s", prComment.Revision, prComment.ID, run.Contexts.Git.Repository)
	if err := vcsClient.PullRequestComment(ctx, run.Contexts.Git.Repository, prComment); err != nil {
		return err
	}

	return nil
}

func pushNotifications(ctx context.Context, db *gorp.DbMap, event sdk.FullEventV2) error {
	if event.ProjectKey == "" {
		return nil
	}
	notifications, err := notification_v2.LoadAll(ctx, db, event.ProjectKey, gorpmapper.GetOptions.WithDecryption)
	if err != nil {
		return sdk.WrapError(err, "unable to load project %s notifications", event.ProjectKey)
	}
	for _, n := range notifications {
		var canSend bool
		if len(n.Filters) == 0 {
			canSend = true
		} else {
		filterLoop:
			for _, f := range n.Filters {
				for _, evt := range f.Events {
					reg, err := regexp.Compile(evt)
					if err != nil {
						log.ErrorWithStackTrace(ctx, err)
						continue
					}
					if reg.MatchString(string(event.Type)) {
						canSend = true
						break filterLoop
					}
				}
			}
		}
		if !canSend {
			continue
		}

		bts, _ := json.Marshal(event)
		req, err := http.NewRequest("POST", n.WebHookURL, bytes.NewBuffer(bts))
		if err != nil {
			log.Error(ctx, "unable to create request for notification %s for project %s: %v", n.Name, n.ProjectKey, err)
			continue
		}
		for k, v := range n.Auth.Headers {
			req.Header.Set(k, v)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Error(ctx, "unable to send notification %s for project %s: %v", n.Name, n.ProjectKey, err)
			continue
		}
		if resp.StatusCode >= 400 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error(ctx, "unable to read body %s: %v", string(body), err)
			}
			log.Error(ctx, "unable to send notification %s for project %s. Http code: %d Body: %s", n.Name, n.ProjectKey, resp.StatusCode, string(body))
			_ = resp.Body.Close()
			continue
		}
		log.Debug(ctx, "notification %s - %s send on event %s", n.ProjectKey, n.Name, event.Type)

	}
	return nil
}

func pushToElasticSearch(ctx context.Context, db *gorp.DbMap, e sdk.FullEventV2) error {
	esServices, err := services.LoadAllByType(ctx, db, sdk.TypeElasticsearch)
	if err != nil {
		return sdk.WrapError(err, "unable to load elasticsearch service")
	}

	if len(esServices) == 0 {
		return nil
	}

	e.Payload = nil
	log.Info(ctx, "sending event %q to %s services", e.Type, sdk.TypeElasticsearch)
	_, code, err := services.NewClient(esServices).DoJSONRequest(context.Background(), "POST", "/v2/events", e, nil)
	if code >= 400 || err != nil {
		return sdk.WrapError(err, "unable to send event %s to elasticsearch [%d]: %v", e.Type, code, err)
	}
	return nil
}
