package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	url2 "net/url"
	"strings"
	"time"

	"github.com/fsamin/go-dump"
	"golang.org/x/crypto/ssh"

	"github.com/ovh/cds/engine/api/cache"
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
	m, err := json.Marshal(g)
	if err != nil {
		return sdk.WrapError(err, "unable to marshal task")
	}
	d.store.SetAdd(cache.Key(gerritRepoKey, vcsServer, repo), g.UUID, string(m))
	return nil
}

// FindGerritTasksByRepo get all gerrit hooks on the given repository
func (d *dao) FindGerritTasksByRepo(vcsServer string, repo string) ([]gerritTaskInfo, error) {
	key := cache.Key(gerritRepoKey, vcsServer, repo)
	nbGerritHooks := d.store.SetCard(key)

	hooks := make([]*gerritTaskInfo, nbGerritHooks, nbGerritHooks)
	for i := 0; i < nbGerritHooks; i++ {
		hooks[i] = &gerritTaskInfo{}
	}
	if err := d.store.SetScan(rootKey, sdk.InterfaceSlice(hooks)...); err != nil {
		return nil, sdk.WrapError(err, "Unable to scan %s", rootKey)
	}

	allHooks := make([]gerritTaskInfo, nbGerritHooks)
	for i := 0; i < nbGerritHooks; i++ {
		allHooks[i] = *hooks[i]
	}

	return allHooks, nil
}

func (s *Service) startGerritHookTask(t *sdk.Task) (*sdk.TaskExecution, error) {
	g := gerritTaskInfo{
		UUID:   t.UUID,
		Events: strings.Split(t.Config[sdk.HookConfigEventFilter].Value, ";"),
	}
	return nil, s.Dao.RegisterGerritRepoHook(t.Config[sdk.HookConfigVCSServer].Value, t.Config[sdk.HookConfigRepoFullName].Value, g)
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
	if gerritEvent.Change != nil {
		payload["git.author"] = gerritEvent.Change.Owner.Username
		payload["git.author.email"] = gerritEvent.Change.Owner.Email
		payload["git.branch"] = gerritEvent.Change.Branch
		payload["git.repository"] = gerritEvent.Change.Project

		payload["cds.triggered_by.username"] = gerritEvent.Change.Owner.Username
		payload["cds.triggered_by.fullname"] = gerritEvent.Change.Owner.Name
		payload["cds.triggered_by.email"] = gerritEvent.Change.Owner.Email

		payload["git.message"] = gerritEvent.Change.CommitMessage
	}
	payload["payload"] = string(e.GerritEvent.Message)

	//payload["git.hash.before"] = pushEvent.Before
	//payload["git.hash"] = pushEvent.After

	d := dump.NewDefaultEncoder(&bytes.Buffer{})
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

			hooks, err := s.Dao.FindGerritTasksByRepo(vcsServer, repo)
			if err != nil {
				log.Error("ComputeGerritStreamEvent > Unable to list task for repo %s/%s", vcsServer, repo)
			}

			msg, err := json.Marshal(e)
			if err != nil {
				log.Error("unable to marshal gerrit event: %v", err)
			}

			for _, h := range hooks {
				if !sdk.IsInArray(e.Type, h.Events) {
					continue
				}

				//Load the task
				gerritHook := s.Dao.FindTask(h.UUID)
				if gerritHook == nil {
					log.Error("Unknown uuid %s", h.UUID)
					continue
				}

				//Prepare a web hook execution
				exec := &sdk.TaskExecution{
					Timestamp: time.Now().UnixNano(),
					UUID:      h.UUID,
					GerritEvent: &sdk.GerritEventExecution{
						Message: msg,
					},
				}

				//Save the web hook execution
				s.Dao.SaveTaskExecution(exec)

				//Push the webhook execution in the queue, so it will be executed
				s.Dao.EnqueueTaskExecution(exec)
			}
		}
	}
}

// ListenGerritStreamEvent listent the gerrit event stream
func ListenGerritStreamEvent(ctx context.Context, v sdk.VCSConfiguration, gerritEventChan chan<- GerritEvent) {
	signer, err := ssh.ParsePrivateKey([]byte(v.Password))
	if err != nil {
		log.Error("unable to read ssh key: %v", err)
	}

	// Create config
	config := &ssh.ClientConfig{
		User: v.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	url, _ := url2.Parse(v.URL)

	// Dial TCP
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", url.Hostname(), v.SSHPort), config)
	if err != nil {
		log.Error("unable to open ssh connection to gerrit: %v", err)
		return
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		log.Error("unable to create new session: %v", err)
		return
	}

	bufferOut := &bytes.Buffer{}
	bufferErr := &bytes.Buffer{}
	session.Stdout = bufferOut
	session.Stderr = bufferErr

	go func() {
		// Run command
		log.Debug("Listening to gerrit event stream %s", v.URL)
		if err := session.Run("gerrit stream-events"); err != nil {
			log.Error("unable to run gerrit stream-events command: %v", err)
		}
	}()

	tick := time.NewTicker(50 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			session.Close()
			conn.Close()
		case <-tick.C:
			if bufferOut.Len() != 0 {
				events := strings.Split(string(bufferOut.Bytes()), "\n")
				for _, e := range events {
					if e == "" {
						continue
					}
					var event GerritEvent
					if err := json.Unmarshal([]byte(e), &event); err != nil {
						log.Error("unable to read gerrit event %v: %s", err, e)
						continue
					}
					gerritEventChan <- event
				}
				bufferOut.Reset()
			}
		default:
		}
	}

}
