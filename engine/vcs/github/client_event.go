package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// ErrNoNewEvents for no new events
var (
	ErrNoNewEvents = fmt.Errorf("No new events")
	ErrNoHook      = fmt.Errorf("No Hook")
)

//GetEvents calls Github et returns GithubEvents as []interface{}
func (g *githubClient) GetEvents(fullname string, dateRef time.Time) ([]interface{}, time.Duration, error) {
	log.Debug("githubClient.GetEvents> loading events for %s after %v", fullname, dateRef)
	var events = []interface{}{}

	interval := 60 * time.Second

	status, body, headers, err := g.get("/repos/" + fullname + "/events")
	if err != nil {
		log.Warning("githubClient.GetEvents> Error %s", err)
		return nil, interval, err
	}

	if status >= http.StatusBadRequest {
		err := sdk.NewError(sdk.ErrUnknownError, errorAPI(body))
		log.Warning("githubClient.GetEvents> Error http %s", err)
		return nil, interval, err
	}

	if status == http.StatusNotModified {
		return nil, interval, ErrNoNewEvents
	}

	nextEvents := []Event{}
	if err := json.Unmarshal(body, &nextEvents); err != nil {
		log.Warning("githubClient.GetEvents> Unable to parse github events: %s", err)
		return nil, interval, fmt.Errorf("Unable to parse github events %s: %s", string(body), err)
	}

	log.Debug("githubClient.GetEvents> Found %d events...")
	//Check here only events after the reference date and only of type PushEvent or CreateEvent
	for _, e := range nextEvents {
		var skipEvent bool
		if e.CreatedAt.After(dateRef) {
			for i := range events {
				e1 := events[i].(Event)
				if e.Payload.Ref == e1.Payload.Ref {
					if e.Type == "DeleteEvent" && e1.Type == "CreateEvent" {
						//Delete event after create event
						if e.CreatedAt.After(e1.CreatedAt.Time) {
							skipEvent = true
						} else {
							//Avoid delete
							events = append(events[:i], events[i+1:]...)
						}
						break
					} else if e.Type == "CreateEvent" && e1.Type == "DeleteEvent" {
						//Delete event before create event
						if e.CreatedAt.After(e1.CreatedAt.Time) {
							events = append(events[:i], events[i+1:]...)
						}
						break
					}
				}
			}

			if e.Type == "PullRequestEvent" {
				switch e.Payload.Action {
				case "opened", "edited", "reopened":
					skipEvent = false
				default:
					skipEvent = true
				}
			}

			if !skipEvent {
				events = append(events, e)
			}
		}
	}

	//Check poll interval
	if headers.Get("X-Poll-Interval") != "" {
		f, err := strconv.ParseFloat(headers.Get("X-Poll-Interval"), 64)
		if err == nil {
			interval = time.Duration(f) * time.Second
		}
	}

	return events, interval, nil
}

//PushEvents returns push events as commits
func (g *githubClient) PushEvents(fullname string, iEvents []interface{}) ([]sdk.VCSPushEvent, error) {
	events := Events{}
	//Cast all the events
	for _, i := range iEvents {

		e := Event{}
		if err := mapstructure.Decode(i, &e); err != nil {
			return nil, err
		}
		if e.Type == "PushEvent" {
			events = append(events, e)
		}
	}

	lastCommitPerBranch := map[string]sdk.VCSCommit{}
	for _, e := range events {
		branch := strings.Replace(e.Payload.Ref, "refs/heads/", "", 1)
		for _, c := range e.Payload.Commits {
			commit := sdk.VCSCommit{
				Hash:      c.Sha,
				Message:   c.Message,
				Timestamp: e.CreatedAt.Unix() * 1000,
				URL:       c.URL,
				Author: sdk.VCSAuthor{
					DisplayName: c.Author.Name,
					Email:       c.Author.Email,
					Name:        e.Actor.DisplayLogin,
					Avatar:      e.Actor.AvatarURL,
				},
			}
			l, b := lastCommitPerBranch[branch]
			if !b || l.Timestamp < commit.Timestamp {
				lastCommitPerBranch[branch] = commit
				continue
			}
		}
	}

	res := []sdk.VCSPushEvent{}
	for b, c := range lastCommitPerBranch {
		branch, err := g.Branch(fullname, b)
		if err != nil || branch == nil {
			log.Warning("githubClient.PushEvents> Unable to find branch %s in %s : %s", b, fullname, err)
			continue
		}
		res = append(res, sdk.VCSPushEvent{
			Branch: *branch,
			Commit: c,
			Repo:   fullname,
		})
	}

	return res, nil
}

//CreateEvents checks create events from a event list
func (g *githubClient) CreateEvents(fullname string, iEvents []interface{}) ([]sdk.VCSCreateEvent, error) {
	events := Events{}
	//Cast all the events
	for _, i := range iEvents {
		e := Event{}
		if err := mapstructure.Decode(i, &e); err != nil {
			return nil, err
		}
		if e.Type == "CreateEvent" {
			events = append(events, e)
		}
	}

	res := []sdk.VCSCreateEvent{}
	for _, e := range events {
		b := e.Payload.Ref
		branch, err := g.Branch(fullname, b)
		if err != nil || branch == nil {
			errtxt := fmt.Sprintf("githubClient.CreateEvents> Unable to find branch %s in %s : %s", b, fullname, err)
			if err != nil && !strings.Contains(errtxt, "Branch not found") {
				log.Warning(errtxt)
			} else {
				log.Debug(errtxt)
			}
			continue
		}
		event := sdk.VCSCreateEvent{
			Branch: *branch,
		}

		c, err := g.Commit(fullname, branch.LatestCommit)
		if err != nil {
			log.Warning("githubClient.CreateEvents> Unable to find commit %s in %s : %s", branch.LatestCommit, fullname, err)
			continue
		}
		event.Commit = c

		res = append(res, event)
	}

	log.Debug("githubClient.CreateEvents> found %d create events : %#v", len(res), res)

	return res, nil
}

//DeleteEvents checks delete events from a event list
func (g *githubClient) DeleteEvents(fullname string, iEvents []interface{}) ([]sdk.VCSDeleteEvent, error) {
	events := Events{}
	//Cast all the events
	for _, i := range iEvents {
		e := Event{}
		if err := mapstructure.Decode(i, &e); err != nil {
			return nil, err
		}
		if e.Type == "DeleteEvent" {
			events = append(events, e)
		}
	}

	res := []sdk.VCSDeleteEvent{}
	for _, e := range events {
		event := sdk.VCSDeleteEvent{
			Branch: sdk.VCSBranch{
				DisplayID: e.Payload.Ref,
			},
		}
		res = append(res, event)
	}

	log.Debug("githubClient.DeleteEvents> found %d delete events : %#v", len(res), res)
	return res, nil
}

//PullRequestEvents checks pull request events from a event list
func (g *githubClient) PullRequestEvents(fullname string, iEvents []interface{}) ([]sdk.VCSPullRequestEvent, error) {
	events := Events{}
	//Cast all the events
	for _, i := range iEvents {
		e := Event{}
		if err := mapstructure.Decode(i, &e); err != nil {
			return nil, err
		}
		if e.Type == "PullRequestEvent" {
			events = append(events, e)
		}
	}

	res := []sdk.VCSPullRequestEvent{}
	for _, e := range events {
		if e.Payload.PullRequest.State != "open" {
			continue
		}
		event := sdk.VCSPullRequestEvent{
			Action: e.Payload.Action,
			Repo:   e.Payload.PullRequest.Head.Repo.FullName,
			Head: sdk.VCSPushEvent{
				Branch: sdk.VCSBranch{
					ID:           e.Payload.PullRequest.Head.Ref,
					DisplayID:    e.Payload.PullRequest.Head.Ref,
					LatestCommit: e.Payload.PullRequest.Head.Sha,
				},
				Commit: sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:        e.Payload.PullRequest.Head.User.Name,
						DisplayName: e.Payload.PullRequest.Head.User.Login,
						Email:       e.Payload.PullRequest.Head.User.Email,
					},
					Hash:    e.Payload.PullRequest.Head.Sha,
					Message: e.Payload.PullRequest.Head.Label,
				},
				CloneURL: e.Payload.PullRequest.Head.Repo.CloneURL,
				Repo:     e.Payload.PullRequest.Head.Repo.FullName,
			},
			Base: sdk.VCSPushEvent{
				Branch: sdk.VCSBranch{
					ID:           e.Payload.PullRequest.Base.Ref,
					DisplayID:    e.Payload.PullRequest.Base.Ref,
					LatestCommit: e.Payload.PullRequest.Base.Sha,
				},
				Commit: sdk.VCSCommit{
					Author: sdk.VCSAuthor{
						Name:        e.Payload.PullRequest.Base.User.Name,
						DisplayName: e.Payload.PullRequest.Base.User.Login,
						Email:       e.Payload.PullRequest.Base.User.Email,
					},
					Hash:    e.Payload.PullRequest.Base.Sha,
					Message: e.Payload.PullRequest.Base.Label,
				},
				CloneURL: e.Payload.PullRequest.Base.Repo.CloneURL,
				Repo:     e.Payload.PullRequest.Base.Repo.FullName,
			},
		}
		res = append(res, event)
	}

	log.Debug("githubClient.PullRequestEvents> found %d pull request events : %#v", len(res), res)

	return res, nil
}
