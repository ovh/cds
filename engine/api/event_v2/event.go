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
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdsclient"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/notification_v2"
	"github.com/ovh/cds/engine/api/repositoriesmanager"
	"github.com/ovh/cds/engine/api/services"
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
		var e sdk.FullEventV2
		if err := store.DequeueWithContext(ctx, eventQueue, 50*time.Millisecond, &e); err != nil {
			ctx := sdk.ContextWithStacktrace(ctx, err)
			log.Error(ctx, "EventV2.DequeueEvent> store.DequeueWithContext err: %v", err)
			continue
		}

		wg := sync.WaitGroup{}

		// Push to elasticsearch
		wg.Add(1)
		goroutines.Exec(ctx, "event.pushToElasticSearch", func(ctx context.Context) {
			defer wg.Done()
			if err := pushToElasticSearch(ctx, db, e); err != nil {
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
			pushToWebsockets(ctx, store, e)
		})

		// Project notifications
		wg.Add(1)
		goroutines.Exec(ctx, "event.notifications", func(ctx context.Context) {
			defer wg.Done()
			if err := pushNotifications(ctx, db, e); err != nil {
				log.Error(ctx, "EventV2.pushNotifications: %v", err)
			}
		})

		// Workflow notifications
		wg.Add(1)
		goroutines.Exec(ctx, "event.workflow.notifications", func(ctx context.Context) {
			defer wg.Done()
			if err := workflowNotifications(ctx, db, store, e, cdsUIURL); err != nil {
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
	// EventRunEnded
	if event.Type != sdk.EventRunEnded || event.ProjectKey == "" {
		return nil
	}

	//event.Payload contains a V2WorkflowRun
	var run sdk.V2WorkflowRun
	if err := json.Unmarshal(event.Payload, &run); err != nil {
		return sdk.WrapError(err, "cannot read payload for type %q", event.Type)
	}

	// always send build status on workflow End
	vcsClient, err := repositoriesmanager.AuthorizedClient(ctx, db, store, event.ProjectKey, event.VCSName)
	if err != nil {
		return sdk.WrapError(err, "can't get AuthorizedClient for %v/%v", event.ProjectKey, event.VCSName)
	}

	buildStatus := sdk.VCSBuildStatus{
		Description:        run.WorkflowName + ":" + run.Status,
		URLCDS:             fmt.Sprintf("%s/project/%s/workflow/%s/run/%d", cdsUIURL, event.ProjectKey, run.WorkflowName, event.RunNumber),
		Context:            fmt.Sprintf("%s-%s", event.ProjectKey, run.WorkflowName),
		Status:             event.Status,
		RepositoryFullname: event.Repository,
		GitHash:            run.Contexts.Git.Ref,
	}
	if err := vcsClient.SetStatus(ctx, buildStatus); err != nil {
		return sdk.WrapError(err, "can't send the build status for %v/%v", event.ProjectKey, event.VCSName)
	}

	if run.WorkflowData.Workflow.On == nil {
		return nil
	}

	if run.WorkflowData.Workflow.On.PullRequest != nil {
		if run.WorkflowData.Workflow.On.PullRequest.Comment != "" {

			err := sendVCSPullRequestComment(ctx, vcsClient, run, run.WorkflowData.Workflow.On.PullRequest.Comment)
			if err != nil {
				return sdk.WrapError(err, "can't send the pull-request comment for %v/%v", event.ProjectKey, event.VCSName)
			}
		}
	}

	if run.WorkflowData.Workflow.On.PullRequestComment != nil {
		if run.WorkflowData.Workflow.On.PullRequestComment.Comment != "" {
			err := sendVCSPullRequestComment(ctx, vcsClient, run, run.WorkflowData.Workflow.On.PullRequestComment.Comment)
			if err != nil {
				return sdk.WrapError(err, "can't send the pull-request comment for %v/%v", event.ProjectKey, event.VCSName)
			}
		}
	}

	return nil
}

func sendVCSPullRequestComment(ctx context.Context, vcsClient sdk.VCSAuthorizedClientService, run sdk.V2WorkflowRun, comment string) error {
	prComment := sdk.VCSPullRequestCommentRequest{
		Revision: run.Contexts.Git.Ref,
		Message:  comment,
	}

	//Check if this branch and this commit is a pullrequest
	prs, err := vcsClient.PullRequests(ctx, run.Contexts.Git.Repository)
	if err != nil {
		log.ErrorWithStackTrace(ctx, err)
		return err
	}

	// Send comment on pull request
	for _, pr := range prs {
		if pr.Head.Branch.DisplayID == run.Contexts.Git.RefName && sdk.VCSIsSameCommit(pr.Head.Branch.LatestCommit, run.Contexts.Git.Ref) && !pr.Merged && !pr.Closed {
			prComment.ID = pr.ID
			log.Info(ctx, "send comment (revision: %v pr: %v) on repo %s", prComment.Revision, prComment.ID, run.Contexts.Git.Repository)
			if err := vcsClient.PullRequestComment(ctx, run.Contexts.Git.Repository, prComment); err != nil {
				return err
			}
			break
		} else {
			log.Info(ctx, "nothing to do on pr %+v for ref %s", pr, run.Contexts.Git.Ref)
		}
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
		canSend := false
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
					if reg.MatchString(event.Type) {
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
