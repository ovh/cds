package hooks

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	"golang.org/x/crypto/ssh"

	"github.com/ovh/cds/engine/cache"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// GerritTaskInfo represents gerrit hook task information and filter
type gerritTaskInfo struct {
	UUID   string   `json:"uuid"`
	Events []string `json:"events"`
}

// RegisterGerritRepoHook register hook on gerrit repository
func (d *dao) RegisterGerritRepoHook(vcsServer string, repo string, g gerritTaskInfo) error {
	return d.store.SetAdd(cache.Key(gerritRepoKey, vcsServer, repo), g.UUID, g)
}

func (d *dao) RemoveGerritRepoHook(vcsServer string, repo string, g gerritTaskInfo) {
	d.store.SetRemove(cache.Key(gerritRepoKey, vcsServer, repo), g.UUID, g)
}

// FindGerritTasksByRepo get all gerrit hooks on the given repository
func (d *dao) FindGerritTasksByRepo(ctx context.Context, vcsServer string, repo string) ([]gerritTaskInfo, error) {
	key := cache.Key(gerritRepoKey, vcsServer, repo)
	nbGerritHooks, err := d.store.SetCard(key)
	if err != nil {
		return nil, sdk.WrapError(err, "unable to setCard %v", key)
	}

	hooks := make([]*gerritTaskInfo, nbGerritHooks)
	for i := 0; i < nbGerritHooks; i++ {
		hooks[i] = &gerritTaskInfo{}
	}
	if err := d.store.SetScan(ctx, key, sdk.InterfaceSlice(hooks)...); err != nil {
		return nil, sdk.WrapError(err, "Unable to scan %s", key)
	}

	allHooks := make([]gerritTaskInfo, nbGerritHooks)
	for i := 0; i < nbGerritHooks; i++ {
		allHooks[i] = *hooks[i]
	}

	return allHooks, nil
}

func (s *Service) startGerritHookTask(t *sdk.Task) error {
	g := gerritTaskInfo{
		UUID:   t.UUID,
		Events: strings.Split(t.Config[sdk.HookConfigEventFilter].Value, ";"),
	}
	s.Dao.RegisterGerritRepoHook(t.Config[sdk.HookConfigVCSServer].Value, t.Config[sdk.HookConfigRepoFullName].Value, g)

	// Check that stream is open
	if _, has := gerritRepoHooks[t.Config[sdk.HookConfigVCSServer].Value]; !has {
		// Start listening to gerrit event stream
		vcsConfig, err := s.Client.VCSConfiguration()
		if err != nil {
			return sdk.WrapError(err, "unable to get vcs configuration")
		}
		s.initGerritStreamEvent(context.Background(), t.Config[sdk.HookConfigVCSServer].Value, vcsConfig)
	}
	return nil
}

func (s *Service) stopGerritHookTask(t *sdk.Task) {
	g := gerritTaskInfo{
		UUID:   t.UUID,
		Events: strings.Split(t.Config[sdk.HookConfigEventFilter].Value, ";"),
	}
	s.Dao.RemoveGerritRepoHook(t.Config[sdk.HookConfigVCSServer].Value, t.Config[sdk.HookConfigRepoFullName].Value, g)
}

func (s *Service) doGerritExecution(e *sdk.TaskExecution) (*sdk.WorkflowNodeRunHookEvent, error) {
	log.Debug("Hooks> Processing gerrit event %s %s", e.UUID, e.Type)

	// Prepare a struct to send to CDS API
	h := sdk.WorkflowNodeRunHookEvent{
		WorkflowNodeHookUUID: e.UUID,
	}

	var gerritEvent GerritEvent
	if err := json.Unmarshal(e.GerritEvent.Message, &gerritEvent); err != nil {
		return nil, sdk.WrapError(err, "unable to unmarshal gerrit event %s", string(e.GerritEvent.Message))
	}

	payload := make(map[string]interface{})
	payload[GIT_EVENT] = gerritEvent.Type

	// assignee-* / change-* / comment-* / draft-* / hashtags-* / patchset-* / reviewer-* / topic-* / vote-*
	if gerritEvent.Change != nil {
		payload[GIT_AUTHOR] = gerritEvent.Change.Owner.Username
		payload[GIT_AUTHOR_EMAIL] = gerritEvent.Change.Owner.Email
		payload[GIT_REPOSITORY] = gerritEvent.Change.Project
		payload[CDS_TRIGGERED_BY_USERNAME] = gerritEvent.Change.Owner.Username
		payload[CDS_TRIGGERED_BY_FULLNAME] = gerritEvent.Change.Owner.Name
		payload[CDS_TRIGGERED_BY_EMAIL] = gerritEvent.Change.Owner.Email

		payload[GIT_MESSAGE] = gerritEvent.Change.CommitMessage
		payload["gerrit.change.id"] = fmt.Sprintf("%s~%s~%s", url.QueryEscape(gerritEvent.Change.Project), url.QueryEscape(gerritEvent.Change.Branch), gerritEvent.Change.ID)
		payload["gerrit.change.url"] = gerritEvent.Change.URL
		payload["gerrit.change.status"] = gerritEvent.Change.Status
		payload["gerrit.change.branch"] = gerritEvent.Change.Branch
	}

	// ref-updated
	if gerritEvent.RefUpdate != nil {
		payload[GIT_HASH_BEFORE] = gerritEvent.RefUpdate.OldRev
		payload[GIT_HASH] = gerritEvent.RefUpdate.NewRev
		payload["gerrit.ref.name"] = gerritEvent.RefUpdate.RefName
	}
	// change-merged / ref-updated
	if gerritEvent.Submitter != nil {
		if gerritEvent.Submitter.Username != "" {
			payload[GIT_AUTHOR] = gerritEvent.Submitter.Username
		}
		if gerritEvent.Submitter.Email != "" {
			payload[GIT_AUTHOR_EMAIL] = gerritEvent.Submitter.Email
		}
	}
	// change-* / comment-* / draft-* / patchset-* / reviewer-* / vote-*
	if gerritEvent.PatchSet != nil {
		payload[GIT_HASH] = gerritEvent.PatchSet.Revision
		if len(gerritEvent.PatchSet.Parents) == 1 {
			payload[GIT_HASH_BEFORE] = gerritEvent.PatchSet.Parents[0]
		}
		payload["gerrit.change.ref"] = gerritEvent.PatchSet.Ref
		if gerritEvent.PatchSet.Author != nil {
			if gerritEvent.PatchSet.Author.Username != "" {
				payload[GIT_AUTHOR] = gerritEvent.PatchSet.Author.Username
			}
			if gerritEvent.PatchSet.Author.Email != "" {
				payload[GIT_AUTHOR_EMAIL] = gerritEvent.PatchSet.Author.Email
			}
		}
	}
	// change-merged
	if gerritEvent.NewRev != "" {
		payload[GIT_HASH] = gerritEvent.NewRev
	}

	// Comment
	if gerritEvent.Type == GerritEventTypeCommentAdded {
		payload["gerrit.comment"] = gerritEvent.Comment
		if gerritEvent.Author != nil {
			payload["gerrit.comment.author.username"] = gerritEvent.Author.Username
			payload["gerrit.comment.author.name"] = gerritEvent.Author.Name
			payload["gerrit.comment.author.email"] = gerritEvent.Author.Email
		}
	}

	payload["gerrit.type"] = gerritEvent.Type

	payload[PAYLOAD] = string(e.GerritEvent.Message)

	d := dump.NewDefaultEncoder()
	d.ExtraFields.Type = false
	d.ExtraFields.Len = false
	d.ExtraFields.DetailedMap = false
	d.ExtraFields.DetailedStruct = false
	d.Formatters = []dump.KeyFormatterFunc{dump.WithDefaultLowerCaseFormatter()}
	payloadValues, errDump := d.ToStringMap(payload)
	if errDump != nil {
		return nil, sdk.WrapError(errDump, "Cannot dump payload %+v ", payload)
	}
	h.Payload = payloadValues

	return &h, nil
}

func (s *Service) ComputeGerritStreamEvent(ctx context.Context, vcsServer string, gerritEventChan <-chan GerritEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-gerritEventChan:
			var repo string
			switch {
			case e.Change != nil:
				repo = e.Change.Project
			case e.RefUpdate != nil:
				repo = e.RefUpdate.Project
			}

			hooks, err := s.Dao.FindGerritTasksByRepo(ctx, vcsServer, repo)
			if err != nil {
				log.Error(ctx, "ComputeGerritStreamEvent > Unable to list task for repo %s/%s", vcsServer, repo)
				continue
			}

			msg, err := json.Marshal(e)
			if err != nil {
				log.Error(ctx, "unable to marshal gerrit event: %v", err)
			}

			for _, h := range hooks {
				if !sdk.IsInArray(e.Type, h.Events) {
					continue
				}

				//Load the task
				gerritHook := s.Dao.FindTask(ctx, h.UUID)
				if gerritHook == nil {
					log.Error(ctx, "Unknown uuid %s", h.UUID)
					continue
				}

				//Prepare a web hook execution
				exec := &sdk.TaskExecution{
					Timestamp: time.Now().UnixNano(),
					UUID:      h.UUID,
					Status:    TaskExecutionScheduled,
					GerritEvent: &sdk.GerritEventExecution{
						Message: msg,
					},
					Type: gerritHook.Type,
				}

				//Save the web hook execution
				s.Dao.SaveTaskExecution(exec)

				//Push the webhook execution in the queue, so it will be executed
				if err := s.Dao.EnqueueTaskExecution(ctx, exec); err != nil {
					log.Error(ctx, "ComputeGerritStreamEvent > error on EnqueueTaskExecution %v", err)
				}
			}
		}
	}
}

// ListenGerritStreamEvent listen the gerrit event stream
func ListenGerritStreamEvent(ctx context.Context, store cache.Store, goRoutines *sdk.GoRoutines, v sdk.VCSConfiguration, gerritEventChan chan<- GerritEvent) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	signer, err := ssh.ParsePrivateKey([]byte(v.Password))
	if err != nil {
		return sdk.WithStack(err)
	}

	// Create config
	config := &ssh.ClientConfig{
		User: v.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	URL, _ := url.Parse(v.URL)

	// Dial TCP
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", URL.Hostname(), v.SSHPort), config)
	if err != nil {
		return sdk.WithStack(err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return sdk.WithStack(err)
	}
	defer session.Close()

	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()
	session.Stdout = w

	stdoutreader := bufio.NewReader(r)

	goRoutines.Exec(ctx, "gerrit-ssh-run", func(ctx context.Context) {
		// Run command
		log.Debug("Listening to gerrit event stream %s", v.URL)
		if err := session.Run("gerrit stream-events"); err != nil {
			log.Error(ctx, "ListenGerritStreamEvent> unable to run gerrit stream-events command: %v", err)
		}
		cancel()
		r.Close()
		return
	})

	lockKey := cache.Key("gerrit", "event", "lock")
	tick := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			line, errs := stdoutreader.ReadString('\n')
			if errs == io.EOF {
				continue
			}
			if errs != nil {
				log.Warning(ctx, "ListenGerritStreamEvent> unable to read string")
				continue
			}
			if line == "" {
				continue
			}
			var event GerritEvent
			lineBytes := []byte(line)
			if err := json.Unmarshal(lineBytes, &event); err != nil {
				log.Error(ctx, "unable to read gerrit event %v: %s", err, line)
				continue
			}

			// Avoid that 2 hook uservice dispatch the same event
			// Take the lock to dispatch an event
			locked, err := store.Lock(lockKey, time.Minute, 100, 15)
			if err != nil {
				log.Error(ctx, "unable to lock %s: %v", lockKey, err)
			}

			// compute md5
			hasher := md5.New()
			hasher.Write(lineBytes) // nolint
			md5 := hex.EncodeToString(hasher.Sum(nil))

			// check if this event has already been dispatched
			k := cache.Key("gerrit", "event", "id", md5)
			var existString string
			b, _ := store.Get(k, &existString)
			if !b {
				_ = store.SetWithTTL(k, md5, 300)
			}

			// release lock
			if locked {
				if err := store.Unlock(lockKey); err == nil {
					log.Error(ctx, "unable to unlock %s: %v", lockKey, err)
				}
			}

			if !b {
				gerritEventChan <- event
			}
		}
	}

}
