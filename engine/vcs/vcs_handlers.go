package vcs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/vcs/github"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func muxVar(r *http.Request, s string) string {
	vars := mux.Vars(r)
	return vars[s]
}

func (s *Service) getAllVCSServersHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return api.WriteJSON(w, r, s.Cfg.Servers, http.StatusOK)
	}
}

func (s *Service) getVCSServersHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		cfg, ok := s.Cfg.Servers[name]
		if !ok {
			return sdk.ErrNotFound
		}
		return api.WriteJSON(w, r, cfg, http.StatusOK)
	}
}

func (s *Service) getVCSServersHooksHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		cfg, ok := s.Cfg.Servers[name]
		if !ok {
			return sdk.ErrNotFound
		}
		res := struct {
			WebhooksSupported         bool   `json:"webhooks_supported"`
			WebhooksDisabled          bool   `json:"webhooks_disabled"`
			WebhooksCreationSupported bool   `json:"webhooks_creation_supported"`
			WebhooksCreationDisabled  bool   `json:"webhooks_creation_disabled"`
			WebhooksIcon              string `json:"webhooks_icon"`
		}{}

		switch {
		case cfg.Bitbucket != nil:
			res.WebhooksSupported = true
			res.WebhooksDisabled = cfg.Bitbucket.DisableWebHooks
			res.WebhooksIcon = sdk.BitbucketIcon
		case cfg.Github != nil:
			res.WebhooksSupported = false
			res.WebhooksDisabled = cfg.Github.DisableWebHooks
			res.WebhooksIcon = sdk.GitHubIcon
		case cfg.Gitlab != nil:
			res.WebhooksSupported = true
			res.WebhooksDisabled = cfg.Gitlab.DisableWebHooks
			res.WebhooksIcon = sdk.GitlabIcon
		}

		return api.WriteJSON(w, r, res, http.StatusOK)
	}
}

func (s *Service) getVCSServersPollingHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		cfg, ok := s.Cfg.Servers[name]
		if !ok {
			return sdk.ErrNotFound
		}
		res := struct {
			PollingSupported bool `json:"polling_supported"`
			PollingDisabled  bool `json:"polling_disabled"`
		}{}

		switch {
		case cfg.Bitbucket != nil:
			res.PollingSupported = false
			res.PollingDisabled = cfg.Bitbucket.DisablePolling
		case cfg.Github != nil:
			res.PollingSupported = true
			res.PollingDisabled = cfg.Github.DisablePolling
		case cfg.Gitlab != nil:
			res.PollingSupported = false
			res.PollingDisabled = cfg.Gitlab.DisablePolling
		}

		return api.WriteJSON(w, r, res, http.StatusOK)
	}
}

func (s *Service) getAuthorizeHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getAuthorizeHandler> VCS server unavailable")
		}

		token, url, err := consumer.AuthorizeRedirect()
		if err != nil {
			return sdk.WrapError(err, "VCS> getAuthorizeHandler>")
		}

		return api.WriteJSON(w, r, map[string]string{
			"token": token,
			"url":   url,
		}, http.StatusOK)
	}
}

func (s *Service) postAuhorizeHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getAuthorizeHandler> VCS server unavailable")
		}

		body := map[string]string{}
		if err := api.UnmarshalBody(r, &body); err != nil {
			return err
		}

		token, secret, err := consumer.AuthorizeToken(body["token"], body["secret"])
		if err != nil {
			return err
		}

		return api.WriteJSON(w, r, map[string]string{
			"token":  token,
			"secret": secret,
		}, http.StatusOK)
	}
}

func (s *Service) getReposHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getReposHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getReposHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getReposHandler> Unable to get authorized client")
		}

		repos, err := client.Repos()
		if err != nil {
			return sdk.WrapError(err, "VCS> getReposHandler> Unable to get repos")
		}

		return api.WriteJSON(w, r, repos, http.StatusOK)
	}
}

func (s *Service) getRepoHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getRepoHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getRepoHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getRepoHandler> Unable to get authorized client")
		}

		ghRepo, err := client.RepoByFullname(fmt.Sprintf("%s/%s", owner, repo))
		if err != nil {
			return sdk.WrapError(err, "VCS> getRepoHandler> Unable to get repo %s/%s", owner, repo)
		}

		return api.WriteJSON(w, r, ghRepo, http.StatusOK)
	}
}

func (s *Service) getBranchesHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getBranchesHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getBranchesHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getBranchesHandler> Unable to get authorized client")
		}

		branches, err := client.Branches(fmt.Sprintf("%s/%s", owner, repo))
		if err != nil {
			return sdk.WrapError(err, "VCS> getBranchesHandler> Unable to get repo %s/%s branches", owner, repo)
		}
		return api.WriteJSON(w, r, branches, http.StatusOK)
	}
}

func (s *Service) getBranchHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		branch := r.URL.Query().Get("branch")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getBranchHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getBranchHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getBranchHandler> Unable to get authorized client")
		}

		ghBranch, err := client.Branch(fmt.Sprintf("%s/%s", owner, repo), branch)
		if err != nil {
			return sdk.WrapError(err, "VCS> getBranchHandler> Unable to get repo %s/%s branch", owner, repo, branch)
		}
		return api.WriteJSON(w, r, ghBranch, http.StatusOK)
	}
}

func (s *Service) getCommitsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		branch := r.URL.Query().Get("branch")
		since := r.URL.Query().Get("since")
		until := r.URL.Query().Get("until")

		log.Debug("getCommitsHandler>")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getCommitsHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getCommitsHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getCommitsHandler> Unable to get authorized client")
		}

		commits, err := client.Commits(fmt.Sprintf("%s/%s", owner, repo), branch, since, until)
		if err != nil {
			return sdk.WrapError(err, "VCS> getCommitsHandler> Unable to get commits on branch %s of %s/%s commits", branch, owner, repo)
		}
		return api.WriteJSON(w, r, commits, http.StatusOK)
	}
}

func (s *Service) getCommitHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		commit := muxVar(r, "commit")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getCommitHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getCommitHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getCommitHandler> Unable to get authorized client")
		}

		c, err := client.Commit(fmt.Sprintf("%s/%s", owner, repo), commit)
		if err != nil {
			return sdk.WrapError(err, "VCS> getCommitHandler> Unable to get commit %s on %s/%s", commit, owner, repo)
		}
		return api.WriteJSON(w, r, c, http.StatusOK)
	}
}

func (s *Service) getPullRequestsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getPullRequestsHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getPullRequestsHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getPullRequestsHandler> Unable to get authorized client")
		}

		c, err := client.PullRequests(fmt.Sprintf("%s/%s", owner, repo))
		if err != nil {
			return sdk.WrapError(err, "VCS> getPullRequestsHandler> Unable to get pull requests on %s/%s", owner, repo)
		}
		return api.WriteJSON(w, r, c, http.StatusOK)
	}
}

func (s *Service) getEventsHandler() api.Handler {
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

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getEventsHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getEventsHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getEventsHandler> Unable to get authorized client")
		}

		evts, delay, err := client.GetEvents(fmt.Sprintf("%s/%s", owner, repo), dateRef)
		if err != nil && err != github.ErrNoNewEvents {
			return sdk.WrapError(err, "VCS> getEventsHandler> Unable to get events on %s/%s", owner, repo)
		}
		res := struct {
			Events []interface{} `json:"events"`
			Delay  time.Duration `json:"delay"`
		}{
			Events: evts,
			Delay:  delay,
		}
		return api.WriteJSON(w, r, res, http.StatusOK)
	}
}

func (s *Service) postFilterEventsHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		evts := []interface{}{}
		if err := api.UnmarshalBody(r, &evts); err != nil {
			return sdk.WrapError(err, "VCS> postFilterEventsHandler> Unable to read body")
		}

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> postFilterEventsHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> postFilterEventsHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> postFilterEventsHandler> Unable to get authorized client")
		}

		filter := r.URL.Query().Get("filter")

		switch filter {
		case "push":
			events, err := client.PushEvents(fmt.Sprintf("%s/%s", owner, repo), evts)
			if err != nil {
				return sdk.WrapError(err, "VCS> postFilterEventsHandler> Unable to filter push events")
			}
			return api.WriteJSON(w, r, events, http.StatusOK)
		case "create":
			events, err := client.CreateEvents(fmt.Sprintf("%s/%s", owner, repo), evts)
			if err != nil {
				return sdk.WrapError(err, "VCS> postFilterEventsHandler> Unable to filter create events")
			}
			return api.WriteJSON(w, r, events, http.StatusOK)
		case "delete":
			events, err := client.DeleteEvents(fmt.Sprintf("%s/%s", owner, repo), evts)
			if err != nil {
				return sdk.WrapError(err, "VCS> postFilterEventsHandler> Unable to filter delete events")
			}
			return api.WriteJSON(w, r, events, http.StatusOK)
		case "pullrequests":
			events, err := client.PullRequestEvents(fmt.Sprintf("%s/%s", owner, repo), evts)
			if err != nil {
				return sdk.WrapError(err, "VCS> postFilterEventsHandler> Unable to filter pullrequests events")
			}
			return api.WriteJSON(w, r, events, http.StatusOK)
		default:
			return sdk.WrapError(sdk.ErrWrongRequest, "VCS> postFilterEventsHandler> Unrecognized filter")
		}
	}
}

func (s *Service) postStatusHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")

		evt := sdk.Event{}
		if err := api.UnmarshalBody(r, &evt); err != nil {
			return sdk.WrapError(err, "VCS> postStatusHandler> unable to read body")
		}

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> postStatusHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> postStatusHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> postStatusHandler> Unable to get authorized client")
		}

		if err := client.SetStatus(evt); err != nil {
			return sdk.WrapError(err, "VCS> postStatusHandler> Unable to set status")
		}

		return nil
	}
}

func (s *Service) postReleaseHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> postReleaseHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> postReleaseHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> postReleaseHandler> Unable to get authorized client")
		}

		body := struct {
			Tag        string `json:"tag"`
			Title      string `json:"title"`
			Descrition string `json:"description"`
		}{}

		if err := api.UnmarshalBody(r, &body); err != nil {
			return sdk.WrapError(err, "VCS> postReleaseHandler> Unable to read body")
		}

		re, err := client.Release(fmt.Sprintf("%s/%s", owner, repo), body.Tag, body.Title, body.Descrition)
		if err != nil {
			return sdk.WrapError(err, "VCS> postReleaseHandler> Unable to create release")
		}

		return api.WriteJSON(w, r, re, http.StatusOK)
	}
}

func (s *Service) postUploadReleaseFileHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")
		release := muxVar(r, "release")
		artifactName := muxVar(r, "artifactName")

		uploadURL, err := url.QueryUnescape(r.URL.Query().Get("upload_url"))
		if err != nil {
			return err
		}

		if _, err := url.Parse(uploadURL); err != nil {
			return err
		}

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> postReleaseHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> postUploadReleaseFileHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> postUploadReleaseFileHandler> Unable to get authorized client")
		}

		if err := client.UploadReleaseFile(fmt.Sprintf("%s/%s", owner, repo), release, uploadURL, artifactName, r.Body); err != nil {
			return sdk.WrapError(err, "VCS> postUploadReleaseFileHandler> Unable to upload release file")
		}

		return nil
	}
}

func (s *Service) getHookHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		hookURL, err := url.QueryUnescape(r.URL.Query().Get("url"))
		if err != nil {
			return err
		}

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getHookHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> getHookHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> getHookHandler> Unable to get authorized client")
		}

		hook, err := client.GetHook(fmt.Sprintf("%s/%s", owner, repo), hookURL)
		if err != nil {
			return sdk.WrapError(err, "VCS> getHookHandler> Unable to get authorized client")
		}

		return api.WriteJSON(w, r, hook, http.StatusOK)
	}
}

func (s *Service) postHookHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> postHookHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> postHookHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> postHookHandler> Unable to get authorized client")
		}

		body := sdk.VCSHook{}
		if err := api.UnmarshalBody(r, &body); err != nil {
			return sdk.WrapError(err, "VCS> postHookHandler> Unable to read body")
		}

		if err := client.CreateHook(fmt.Sprintf("%s/%s", owner, repo), &body); err != nil {
			return err
		}
		return api.WriteJSON(w, r, body, http.StatusOK)
	}
}

func (s *Service) deleteHookHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		hookURL, err := url.QueryUnescape(r.URL.Query().Get("url"))
		if err != nil {
			return err
		}

		hookID := r.URL.Query().Get("id")

		accessToken, accessTokenSecret, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> deleteHookHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS> deleteHookHandler> VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(accessToken, accessTokenSecret)
		if err != nil {
			return sdk.WrapError(err, "VCS> deleteHookHandler> Unable to get authorized client")
		}

		var hook sdk.VCSHook
		if hookID == "" {
			var err error
			hook, err = client.GetHook(fmt.Sprintf("%s/%s", owner, repo), hookURL)
			if err != nil {
				return sdk.WrapError(err, "VCS> deleteHookHandler> Unable to get hook %s", hookURL)
			}
		} else {
			hook = sdk.VCSHook{
				ID: hookID,
			}
		}

		return client.DeleteHook(fmt.Sprintf("%s/%s", owner, repo), hook)
	}
}
