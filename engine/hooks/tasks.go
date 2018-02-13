package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	"github.com/gorhill/cronexpr"

	"github.com/ovh/cds/engine/api/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

//This are all the types
const (
	TypeRepoManagerWebHook = "RepoWebHook"
	TypeWebHook            = "Webhook"
	TypeScheduler          = "Scheduler"
	TypeRepoPoller         = "RepoPoller"

	GithubHeader    = "X-Github-Event"
	GitlabHeader    = "X-Gitlab-Event"
	BitbucketHeader = "X-Event-Key"
)

var (
	rootKey           = cache.Key("hooks", "tasks")
	executionRootKey  = cache.Key("hooks", "tasks", "executions")
	schedulerQueueKey = cache.Key("hooks", "scheduler", "queue")
)

// runTasks should run as a long-running goroutine
func (s *Service) runTasks(ctx context.Context) error {
	if err := s.synchronizeTasks(); err != nil {
		log.Error("Hook> Unable to synchronize tasks: %v", err)
	}

	if err := s.startTasks(ctx); err != nil {
		log.Error("Hook> Exit running tasks: %v", err)
		return err
	}
	<-ctx.Done()
	return ctx.Err()
}

func (s *Service) synchronizeTasks() error {
	t0 := time.Now()
	defer func() {
		log.Info("Hooks> All tasks has been resynchronized (%.3fs)", time.Since(t0).Seconds())
	}()

	log.Info("Hooks> Synchronizing tasks from CDS API (%s)", s.Cfg.API.HTTP.URL)

	//Get all hooks from CDS, and synchronize the tasks in cache
	hooks, err := s.cds.WorkflowAllHooksList()
	if err != nil {
		return sdk.WrapError(err, "synchronizeTasks> Unable to get hooks")
	}

	allOldTasks, err := s.Dao.FindAllTasks()
	if err != nil {
		return sdk.WrapError(err, "synchronizeTasks> Unable to get allOldTasks")
	}

	//Delete all old task which are not referenced in CDS API anymore
	for i := range allOldTasks {
		t := &allOldTasks[i]
		var found bool
		for _, h := range hooks {
			if h.UUID == t.UUID {
				found = true
				break
			}
		}
		if !found {
			s.Dao.DeleteTask(t)
			log.Info("Hook> Task %s deleted on synchronization", t.UUID)
		}
	}

	for _, h := range hooks {
		confProj := h.Config["project"]
		confWorkflow := h.Config["workflow"]
		if confProj.Value == "" || confWorkflow.Value == "" {
			log.Error("Hook> Unable to synchronize task %+v: %v", h, err)
			continue
		}
		t, err := s.hookToTask(&h)
		if err != nil {
			log.Error("Hook> Unable to synchronize task %+v: %v", h, err)
			continue
		}
		s.Dao.SaveTask(t)
	}

	return nil
}

func (s *Service) hookToTask(h *sdk.WorkflowNodeHook) (*sdk.Task, error) {
	if h.WorkflowHookModel.Type != sdk.WorkflowHookModelBuiltin {
		return nil, fmt.Errorf("Unsupported hook type: %s", h.WorkflowHookModel.Type)
	}

	switch h.WorkflowHookModel.Name {
	case sdk.WebHookModelName:
		h.Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
			Value:        fmt.Sprintf("%s/webhook/%s", s.Cfg.URLPublic, h.UUID),
			Configurable: false,
		}
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeWebHook,
			Config: h.Config,
		}, nil
	case sdk.RepositoryWebHookModelName:
		h.Config["webHookURL"] = sdk.WorkflowNodeHookConfigValue{
			Value:        fmt.Sprintf("%s/webhook/%s", s.Cfg.URLPublic, h.UUID),
			Configurable: false,
		}
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeRepoManagerWebHook,
			Config: h.Config,
		}, nil
	case sdk.SchedulerModelName:
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeScheduler,
			Config: h.Config,
		}, nil
	case sdk.GitPollerModelName:
		return &sdk.Task{
			UUID:   h.UUID,
			Type:   TypeRepoPoller,
			Config: h.Config,
		}, nil
	}

	return nil, fmt.Errorf("Unsupported hook: %s", h.WorkflowHookModel.Name)
}

func (s *Service) startTasks(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	defer cancel()

	//Load all the tasks
	tasks, err := s.Dao.FindAllTasks()
	if err != nil {
		return sdk.WrapError(err, "Hook> startTasks> Unable to find all tasks")
	}

	//Start the tasks
	for i := range tasks {
		t := &tasks[i]
		if err := s.startTask(c, t); err != nil {
			log.Error("Hooks> runLongRunningTasks> Unable to start tasks: %v", err)
			continue
		}
	}
	return nil
}

func (s *Service) startTask(ctx context.Context, t *sdk.Task) error {
	t.Stopped = false
	s.Dao.SaveTask(t)

	switch t.Type {
	case TypeWebHook, TypeRepoManagerWebHook:
		return nil
	case TypeScheduler, TypeRepoPoller:
		return s.prepareNextScheduledTaskExecution(t)
	default:
		return fmt.Errorf("Unsupported task type %s", t.Type)
	}
}

func (s *Service) prepareNextScheduledTaskExecution(t *sdk.Task) error {
	if t.Stopped {
		return nil
	}

	//Load the last execution of this task
	execs, err := s.Dao.FindAllTaskExecutions(t)
	if err != nil {
		return sdk.WrapError(err, "startTask> unable to load last executions")
	}

	//The last execution has not been executed, let it go
	if len(execs) > 0 && execs[len(execs)-1].ProcessingTimestamp == 0 {
		log.Debug("Hooks> Scheduled tasks %s ready. Next execution scheduled on %v", t.UUID, time.Unix(0, execs[len(execs)-1].Timestamp))
		return nil
	}

	//Load the location for the timezone
	confTimezone := t.Config["timezone"]
	loc, err := time.LoadLocation(confTimezone.Value)
	if err != nil {
		return sdk.WrapError(err, "startTask> unable to parse timezone: %s", t.Config["timezone"])
	}

	var exec *sdk.TaskExecution
	var nextSchedule time.Time
	switch t.Type {
	case TypeScheduler:
		//Parse the cron expr
		confCron := t.Config["cron"]
		cronExpr, err := cronexpr.Parse(confCron.Value)
		if err != nil {
			return sdk.WrapError(err, "startTask> unable to parse cron expression: %s", t.Config["cron"])
		}

		//Compute a new date
		t0 := time.Now().In(loc)
		nextSchedule = cronExpr.Next(t0)

	case TypeRepoPoller:
		nextSchedule = time.Now().Add(time.Minute * 2)
		if val, ok := t.Config["next_execution"]; ok {
			nextExec, errT := strconv.ParseInt(val.Value, 10, 64)
			if errT == nil {
				nextSchedule = time.Unix(nextExec, 0)
			}
		}
	}

	//Craft a new execution
	exec = &sdk.TaskExecution{
		Timestamp: nextSchedule.UnixNano(),
		Type:      t.Type,
		UUID:      t.UUID,
		Config:    t.Config,
		ScheduledTask: &sdk.ScheduledTaskExecution{
			DateScheduledExecution: fmt.Sprintf("%v", nextSchedule),
		},
	}

	s.Dao.SaveTaskExecution(exec)
	//We don't push in queue, we will the scheduler to run it

	log.Debug("Hooks> Scheduled tasks %v ready. Next execution scheduled on %v", t.UUID, time.Unix(0, exec.Timestamp))

	return nil
}

func (s *Service) stopTask(ctx context.Context, t *sdk.Task) error {
	log.Info("Hooks> Stopping task %s", t.UUID)
	t.Stopped = true
	s.Dao.SaveTask(t)

	switch t.Type {
	case TypeWebHook, TypeScheduler, TypeRepoManagerWebHook:
		log.Debug("Hooks> Tasks %s has been stopped", t.UUID)
		return nil
	default:
		return fmt.Errorf("Unsupported task type %s", t.Type)
	}
}

func (s *Service) doTask(ctx context.Context, t *sdk.Task, e *sdk.TaskExecution) error {
	if t.Stopped {
		return nil
	}

	var hs []sdk.WorkflowNodeRunHookEvent
	var h *sdk.WorkflowNodeRunHookEvent
	var err error

	switch {
	case e.WebHook != nil:
		h, err = s.doWebHookExecution(e)
	case e.ScheduledTask != nil && e.Type == TypeScheduler:
		h, err = s.doScheduledTaskExecution(e)
	case e.ScheduledTask != nil && e.Type == TypeRepoPoller:
		//Populate next execution
		hs, err = s.doPollerTaskExecution(t, e)
	default:
		err = fmt.Errorf("Unsupported task type %s", e.Type)
	}

	if err != nil {
		return err
	}
	if h != nil {
		hs = append(hs, *h)
	}
	if hs == nil || len(hs) == 0 {
		return nil
	}

	// Call CDS API
	confProj := t.Config["project"]
	confWorkflow := t.Config["workflow"]
	var globalErr error
	for _, hEvent := range hs {
		run, err := s.cds.WorkflowRunFromHook(confProj.Value, confWorkflow.Value, hEvent)
		if err != nil {
			globalErr = err
			log.Error("Hooks> Unable to run workflow %s", err)
		}

		//Save the run number
		e.WorkflowRun = run.Number
		log.Debug("Hooks> workflow %s/%s#%d has been triggered", t.Config["project"], t.Config["workflow"], run.Number)
	}

	if globalErr != nil {
		return sdk.WrapError(globalErr, "Hooks> Unable to run workflow")
	}

	return nil
}

func (s *Service) doPollerTaskExecution(task *sdk.Task, taskExec *sdk.TaskExecution) ([]sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing polling task %s", taskExec.UUID)

	tExecs, errF := s.Dao.FindAllTaskExecutions(task)
	if errF != nil {
		return nil, errF
	}

	var maxTs int64
	for _, tExec := range tExecs {
		if tExec.Status == TaskExecutionDone && maxTs < tExec.Timestamp {
			maxTs = tExec.Timestamp
		}
	}
	workflowID, errP := strconv.ParseInt(taskExec.Config["workflow_id"].Value, 10, 64)
	if errP != nil {
		return nil, sdk.WrapError(errP, "Hooks> doPollerTaskExecution> Cannot convert workflow id %s", taskExec.Config["workflow_id"].Value)
	}
	events, interval, err := s.cds.PollVCSEvents(taskExec.UUID, workflowID, taskExec.Config["vcsServer"].Value, maxTs/1000)
	if err != nil {
		return nil, sdk.WrapError(err, "Hooks> doPollerTaskExecution> Cannot poll vcs events for workflow %s with vcsserver %s", taskExec.Config["workflow"].Value, taskExec.Config["vcsServer"].Value)
	}

	var hookEvents []sdk.WorkflowNodeRunHookEvent
	if len(events.PushEvents) > 0 || len(events.PullRequestEvents) > 0 {
		i := 0
		hookEvents = make([]sdk.WorkflowNodeRunHookEvent, len(events.PushEvents)+len(events.PullRequestEvents))
		for _, pushEvent := range events.PushEvents {
			payload := fillPayload(pushEvent)
			hookEvents[i] = sdk.WorkflowNodeRunHookEvent{
				WorkflowNodeHookUUID: task.UUID,
				Payload:              payload,
			}
			i++
		}

		for _, pullRequestEvent := range events.PullRequestEvents {
			payload := fillPayload(pullRequestEvent.Head)
			hookEvents[i] = sdk.WorkflowNodeRunHookEvent{
				WorkflowNodeHookUUID: task.UUID,
				Payload:              payload,
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

func (s *Service) doScheduledTaskExecution(t *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing scheduled task %s", t.UUID)

	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: t.UUID,
	}

	//Prepare the payload
	//Anything can be pushed in the configuration, just avoid sending
	payloadValues := map[string]string{}
	if payload, ok := t.Config["payload"]; ok && payload.Value != "{}" {
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
		case "project", "workflow", "cron", "timezone", "payload":
		default:
			payloadValues[k] = v.Value
		}
	}
	payloadValues["cds.triggered_by.username"] = "cds.scheduler"
	payloadValues["cds.triggered_by.fullname"] = "CDS Scheduler"
	h.Payload = payloadValues

	return &h, nil
}

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
		payload["git.branch"] = strings.TrimPrefix(strings.TrimPrefix(pushEvent.Ref, "refs/heads/"), "refs/tags/")
		payload["git.hash.before"] = pushEvent.Before
		payload["git.hash"] = pushEvent.After
		payload["git.repository"] = pushEvent.Repository.FullName
		payload["cds.triggered_by.username"] = pushEvent.HeadCommit.Author.Username
		payload["cds.triggered_by.fullname"] = pushEvent.HeadCommit.Author.Name
		payload["cds.triggered_by.email"] = pushEvent.HeadCommit.Author.Email

		if strings.HasPrefix(pushEvent.Ref, "refs/tags/") {
			payload["git.tag"] = strings.TrimPrefix(pushEvent.Ref, "refs/tags/")
		}
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
		payload["git.branch"] = strings.TrimPrefix(strings.TrimPrefix(pushEvent.Ref, "refs/heads/"), "refs/tags/")
		payload["git.hash.before"] = pushEvent.Before
		payload["git.hash"] = pushEvent.After
		payload["git.repository"] = pushEvent.Repository.Name

		payload["cds.triggered_by.username"] = pushEvent.UserUsername
		payload["cds.triggered_by.fullname"] = pushEvent.UserName
		payload["cds.triggered_by.email"] = pushEvent.UserEmail

		if strings.HasPrefix(pushEvent.Ref, "refs/tags/") {
			payload["git.tag"] = strings.TrimPrefix(pushEvent.Ref, "refs/tags/")
		}
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

		payload["git.branch"] = strings.TrimPrefix(strings.TrimPrefix(pushEvent.Changes[0].RefID, "refs/heads/"), "refs/tags/")
		payload["git.hash.before"] = pushEvent.Changes[0].FromHash
		payload["git.hash"] = pushEvent.Changes[0].ToHash
		payload["git.repository"] = fmt.Sprintf("%s/%s", pushEvent.Repository.Project.Key, pushEvent.Repository.Slug)

		payload["cds.triggered_by.username"] = pushEvent.Actor.Name
		payload["cds.triggered_by.fullname"] = pushEvent.Actor.DisplayName
		payload["cds.triggered_by.email"] = pushEvent.Actor.EmailAddress

		if strings.HasPrefix(pushEvent.Changes[0].RefID, "refs/tags/") {
			payload["git.tag"] = strings.TrimPrefix(pushEvent.Changes[0].RefID, "refs/tags/")
		}
	default:
		log.Warning("executeRepositoryWebHook> Repository manager not found. Cannot read %s", string(t.WebHook.RequestBody))
		return nil, nil
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
	confMethod := t.Config["method"]
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
		ct, _, err = mime.ParseMediaType(ct)
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
		case "project", "workflow", "method":
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
