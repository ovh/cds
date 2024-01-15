package vcs

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/vcs/github"
	"github.com/ovh/cds/sdk"
)

func muxVar(r *http.Request, s string) string {
	vars := mux.Vars(r)
	return vars[s]
}

// QueryString return a string from a query parameter
func QueryString(r *http.Request, s string) string {
	return r.FormValue(s)
}

func (s *Service) getReposHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(err, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client")
		}

		repos, err := client.Repos(ctx)
		if err != nil {
			return sdk.WrapError(err, "Unable to get repos")
		}

		return service.WriteJSON(w, repos, http.StatusOK)
	}
}

func (s *Service) getRepoHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		ghRepo, err := client.RepoByFullname(ctx, fmt.Sprintf("%s/%s", owner, repo))
		if err != nil {
			return sdk.WrapError(err, "Unable to get repo %s/%s", owner, repo)
		}

		return service.WriteJSON(w, ghRepo, http.StatusOK)
	}
}

func (s *Service) getBranchesHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		limitS := r.URL.Query().Get("limit")

		var limit int64
		if limitS != "" {
			l, err := strconv.Atoi(limitS)
			if err != nil {
				return sdk.NewErrorFrom(sdk.ErrInvalidData, "limit must be an integer")
			}
			limit = int64(l)
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		branches, err := client.Branches(ctx, fmt.Sprintf("%s/%s", owner, repo), sdk.VCSBranchesFilter{Limit: limit})
		if err != nil {
			return sdk.WrapError(err, "Unable to get repo %s/%s", owner, repo)
		}
		return service.WriteJSON(w, branches, http.StatusOK)
	}
}

func (s *Service) getBranchHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		branch := r.URL.Query().Get("branch")
		defaultBranchS := r.URL.Query().Get("default")

		var defaultBranch bool
		if defaultBranchS == "true" {
			defaultBranch = true
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		ghBranch, err := client.Branch(ctx, fmt.Sprintf("%s/%s", owner, repo), sdk.VCSBranchFilters{BranchName: branch, Default: defaultBranch})
		if err != nil {
			return sdk.WrapError(err, "Unable to get repo %s/%s branch %s", owner, repo, branch)
		}
		return service.WriteJSON(w, ghBranch, http.StatusOK)
	}
}

func (s *Service) getTagHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		tag := muxVar(r, "tagName")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		vcsTag, err := client.Tag(ctx, fmt.Sprintf("%s/%s", owner, repo), tag)
		if err != nil {
			return sdk.WrapError(err, "Unable to get tag %s on %s/%s", tag, owner, repo)
		}
		return service.WriteJSON(w, vcsTag, http.StatusOK)
	}
}

func (s *Service) getTagsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		log.Debug(ctx, "getTagsHandler>")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		tags, err := client.Tags(ctx, fmt.Sprintf("%s/%s", owner, repo))
		if err != nil {
			return sdk.WrapError(err, "Unable to get tags on %s/%s", owner, repo)
		}
		return service.WriteJSON(w, tags, http.StatusOK)
	}
}

func (s *Service) getCommitsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		branch := r.URL.Query().Get("branch")
		since := r.URL.Query().Get("since")
		until := r.URL.Query().Get("until")

		log.Debug(ctx, "getCommitsHandler>")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		commits, err := client.Commits(ctx, fmt.Sprintf("%s/%s", owner, repo), branch, since, until)
		if err != nil {
			return sdk.WrapError(err, "Unable to get commits on branch %s of %s/%s commits", branch, owner, repo)
		}
		return service.WriteJSON(w, commits, http.StatusOK)
	}
}

func (s *Service) getCommitsBetweenRefsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		base := r.URL.Query().Get("base")
		head := r.URL.Query().Get("head")

		log.Debug(ctx, "getCommitsBetweenRefsHandler>")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		commits, err := client.CommitsBetweenRefs(ctx, fmt.Sprintf("%s/%s", owner, repo), base, head)
		if err != nil {
			return sdk.WrapError(err, "Unable to get commits of %s/%s commits diff between %s and %s", owner, repo, base, head)
		}
		return service.WriteJSON(w, commits, http.StatusOK)
	}
}

func (s *Service) getCommitHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		commit := muxVar(r, "commit")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		c, err := client.Commit(ctx, fmt.Sprintf("%s/%s", owner, repo), commit)
		if err != nil {
			return sdk.WrapError(err, "Unable to get commit %s on %s/%s", commit, owner, repo)
		}
		return service.WriteJSON(w, c, http.StatusOK)
	}
}

func (s *Service) getCommitStatusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		commit := muxVar(r, "commit")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		statuses, err := client.ListStatuses(ctx, fmt.Sprintf("%s/%s", owner, repo), commit)
		if err != nil {
			return sdk.WrapError(err, "Unable to get commit %s statuses on %s/%s", commit, owner, repo)
		}

		return service.WriteJSON(w, statuses, http.StatusOK)
	}
}

func (s *Service) getPullRequestHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		id := muxVar(r, "id")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		c, err := client.PullRequest(ctx, fmt.Sprintf("%s/%s", owner, repo), id)
		if err != nil {
			return sdk.WrapError(err, "Unable to get pull requests on %s/%s", owner, repo)
		}
		return service.WriteJSON(w, c, http.StatusOK)
	}
}

func (s *Service) getPullRequestsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		state := sdk.VCSPullRequestState(QueryString(r, "state"))
		if state != "" && !state.IsValid() {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given pull request state %s", state)
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		c, err := client.PullRequests(ctx, fmt.Sprintf("%s/%s", owner, repo), sdk.VCSPullRequestOptions{
			State: state,
		})
		if err != nil {
			return sdk.WrapError(err, "Unable to get pull requests on %s/%s", owner, repo)
		}
		return service.WriteJSON(w, c, http.StatusOK)
	}
}

func (s *Service) postPullRequestsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		var prRequest sdk.VCSPullRequest
		if err := service.UnmarshalBody(r, &prRequest); err != nil {
			return sdk.WithStack(err)
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		c, err := client.PullRequestCreate(ctx, fmt.Sprintf("%s/%s", owner, repo), prRequest)
		if err != nil {
			return sdk.WrapError(err, "Unable to create pull requests on %s/%s", owner, repo)
		}
		return service.WriteJSON(w, c, http.StatusOK)
	}
}

func (s *Service) postPullRequestCommentHandler() service.Handler {
	return func(ctx context.Context, _ http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		var body sdk.VCSPullRequestCommentRequest
		if err := service.UnmarshalBody(r, &body); err != nil {
			return sdk.WithStack(err)
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		if err := client.PullRequestComment(ctx, fmt.Sprintf("%s/%s", owner, repo), body); err != nil {
			return sdk.WrapError(err, "Unable to create new PR comment %s %s/%s", name, owner, repo)
		}

		return nil
	}
}

func (s *Service) getEventsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		dateRefStr := r.URL.Query().Get("since")
		dateRef := time.Time{}

		if dateRefStr != "" {
			dateRefInt, err := strconv.Atoi(dateRefStr)
			if err != nil {
				return sdk.WrapError(sdk.ErrWrongRequest, "VCS> getEventsHandler>")
			}
			dateRef = time.Unix(int64(dateRefInt), 0)
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		evts, delay, err := client.GetEvents(ctx, fmt.Sprintf("%s/%s", owner, repo), dateRef)
		if err != nil && err != github.ErrNoNewEvents {
			return sdk.WrapError(err, "Unable to get events on %s/%s", owner, repo)
		}
		res := struct {
			Events []interface{} `json:"events"`
			Delay  time.Duration `json:"delay"`
		}{
			Events: evts,
			Delay:  delay,
		}
		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func (s *Service) postFilterEventsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		evts := []interface{}{}
		if err := service.UnmarshalBody(r, &evts); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		filter := r.URL.Query().Get("filter")

		switch filter {
		case "push":
			events, err := client.PushEvents(ctx, fmt.Sprintf("%s/%s", owner, repo), evts)
			if err != nil {
				return sdk.WrapError(err, "Unable to filter push events")
			}
			return service.WriteJSON(w, events, http.StatusOK)
		case "create":
			events, err := client.CreateEvents(ctx, fmt.Sprintf("%s/%s", owner, repo), evts)
			if err != nil {
				return sdk.WrapError(err, "Unable to filter create events")
			}
			return service.WriteJSON(w, events, http.StatusOK)
		case "delete":
			events, err := client.DeleteEvents(ctx, fmt.Sprintf("%s/%s", owner, repo), evts)
			if err != nil {
				return sdk.WrapError(err, "Unable to filter delete events")
			}
			return service.WriteJSON(w, events, http.StatusOK)
		case "pullrequests":
			events, err := client.PullRequestEvents(ctx, fmt.Sprintf("%s/%s", owner, repo), evts)
			if err != nil {
				return sdk.WrapError(err, "Unable to filter pullrequests events")
			}
			return service.WriteJSON(w, events, http.StatusOK)
		default:
			return sdk.WrapError(sdk.ErrWrongRequest, "VCS> postFilterEventsHandler> Unrecognized filter")
		}
	}
}

func (s *Service) postStatusHandler() service.Handler {
	return func(ctx context.Context, _ http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")

		evt := sdk.Event{}
		if err := service.UnmarshalBody(r, &evt); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client")
		}

		var disableStatusDetails bool
		d := r.URL.Query().Get("disableStatusDetails")
		if d != "" {
			disableStatusDetails, _ = strconv.ParseBool(d)
		}

		if err := client.SetStatus(ctx, evt, disableStatusDetails); err != nil {
			return sdk.WrapError(err, "Unable to set status on %s", name)
		}

		return nil
	}
}

func (s *Service) postReleaseHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		body := struct {
			Tag        string `json:"tag"`
			Title      string `json:"title"`
			Descrition string `json:"description"`
		}{}

		if err := service.UnmarshalBody(r, &body); err != nil {
			return sdk.WrapError(err, "Unable to read body")
		}

		re, err := client.Release(ctx, fmt.Sprintf("%s/%s", owner, repo), body.Tag, body.Title, body.Descrition)
		if err != nil {
			return sdk.WrapError(err, "Unable to create release %s %s/%s", name, owner, repo)
		}

		return service.WriteJSON(w, re, http.StatusOK)
	}
}

func (s *Service) postUploadReleaseFileHandler() service.Handler {
	return func(ctx context.Context, _ http.ResponseWriter, r *http.Request) error {
		defer r.Body.Close() // nolint
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		release := muxVar(r, "release")
		artifactName := muxVar(r, "artifactName")

		uploadURL, err := url.QueryUnescape(r.URL.Query().Get("upload_url"))
		if err != nil {
			return sdk.WithStack(err)
		}

		contentLength, err := strconv.Atoi(r.Header.Get("Content-Length"))
		if err != nil {
			return sdk.WithStack(err)
		}

		if _, err := url.Parse(uploadURL); err != nil {
			return sdk.WithStack(err)
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		if err := client.UploadReleaseFile(ctx, fmt.Sprintf("%s/%s", owner, repo), release, uploadURL, artifactName, r.Body, contentLength); err != nil {
			return sdk.WrapError(err, "Unable to upload release file %s %s/%s", name, owner, repo)
		}
		return sdk.WithStack(r.Body.Close())
	}
}

func (s *Service) getHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		hookURL, err := url.QueryUnescape(r.URL.Query().Get("url"))
		if err != nil {
			return err
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		hook, err := client.GetHook(ctx, fmt.Sprintf("%s/%s", owner, repo), hookURL)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		return service.WriteJSON(w, hook, http.StatusOK)
	}
}

func (s *Service) putHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		body := sdk.VCSHook{}
		if err := service.UnmarshalBody(r, &body); err != nil {
			return sdk.WrapError(err, "Unable to read body %s %s/%s", name, owner, repo)
		}

		if err := client.UpdateHook(ctx, fmt.Sprintf("%s/%s", owner, repo), &body); err != nil {
			return sdk.WrapError(err, "Update %s %s/%s", name, owner, repo)
		}
		return service.WriteJSON(w, body, http.StatusOK)
	}
}

func (s *Service) postHookHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		body := sdk.VCSHook{}
		if err := service.UnmarshalBody(r, &body); err != nil {
			return sdk.WrapError(err, "unable to read body %s %s/%s", name, owner, repo)
		}

		if err := client.CreateHook(ctx, fmt.Sprintf("%s/%s", owner, repo), &body); err != nil {
			return sdk.WrapError(err, "cannot create hook on %s for repository %s/%s", name, owner, repo)
		}
		return service.WriteJSON(w, body, http.StatusOK)
	}
}

func (s *Service) deleteHookHandler() service.Handler {
	return func(ctx context.Context, _ http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		hookURL, err := url.QueryUnescape(r.URL.Query().Get("url"))
		if err != nil {
			return err
		}

		hookID := r.URL.Query().Get("id")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		var hook sdk.VCSHook
		if hookID == "" {
			var err error
			hook, err = client.GetHook(ctx, fmt.Sprintf("%s/%s", owner, repo), hookURL)
			if err != nil {
				return sdk.WrapError(err, "Unable to get hook %s", hookURL)
			}
		} else {
			hook = sdk.VCSHook{
				ID: hookID,
			}
		}

		return client.DeleteHook(ctx, fmt.Sprintf("%s/%s", owner, repo), hook)
	}
}

func (s *Service) SearchPullRequestHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		commit := QueryString(r, "commit")
		state := QueryString(r, "state")
		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s", name)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s", name)
		}

		pr, err := client.SearchPullRequest(ctx, owner+"/"+repo, commit, state)
		if err != nil {
			return err
		}
		return service.WriteJSON(w, pr, http.StatusOK)
	}
}

func (s *Service) getListForks() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		forks, err := client.ListForks(ctx, fmt.Sprintf("%s/%s", owner, repo))
		if err != nil {
			return sdk.WrapError(err, "Unable to get forks %s %s/%s", name, owner, repo)
		}

		return service.WriteJSON(w, forks, http.StatusOK)
	}
}

// Status returns sdk.MonitoringStatus, implements interface service.Service
func (s *Service) Status(ctx context.Context) *sdk.MonitoringStatus {
	return s.NewMonitoringStatus()
}

func (s *Service) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, _ *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(ctx), status)
	}
}

func (s *Service) getListContentsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		commit := QueryString(r, "commit")
		filePath, err := url.PathUnescape(muxVar(r, "filePath"))
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to get filepath: %v", err)
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		contents, err := client.ListContent(ctx, owner+"/"+repo, commit, filePath)
		if err != nil {
			return sdk.WrapError(err, "unable to list content")
		}
		return service.WriteJSON(w, contents, http.StatusOK)
	}
}

func (s *Service) getFileContentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		commit := QueryString(r, "commit")

		filePath, err := url.PathUnescape(muxVar(r, "filePath"))
		if err != nil {
			return sdk.NewErrorFrom(sdk.ErrInvalidData, "unable to get filepath: %v", err)
		}

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		content, err := client.GetContent(ctx, owner+"/"+repo, commit, filePath)
		if err != nil {
			return sdk.WrapError(err, "unable to get file content")
		}
		return service.WriteJSON(w, content, http.StatusOK)
	}
}

func (s *Service) archiveHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		vcsAuth, err := getVCSAuth(ctx)
		if err != nil {
			return sdk.WrapError(sdk.ErrUnauthorized, "unable to get access token header")
		}

		consumer, err := s.getConsumer(name, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, vcsAuth)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}

		body := sdk.VCSArchiveRequest{}
		if err := service.UnmarshalBody(r, &body); err != nil {
			return sdk.WrapError(err, "unable to read body %s %s/%s", name, owner, repo)
		}

		reader, header, err := client.GetArchive(ctx, owner+"/"+repo, body.Path, body.Format, body.Commit)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Disposition", header.Get("Content-Disposition"))
		if _, err := io.Copy(w, reader); err != nil {
			return err
		}
		return nil
	}
}
