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
	"github.com/xanzy/go-gitlab"

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

func (s *Service) getVCSGerritHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		servers := make(map[string]sdk.VCSGerritConfiguration, len(s.Cfg.Servers))
		for k, v := range s.Cfg.Servers {
			if v.Gerrit == nil {
				continue
			}
			servers[k] = sdk.VCSGerritConfiguration{
				SSHUsername:   v.Gerrit.EventStream.User,
				SSHPrivateKey: v.Gerrit.EventStream.PrivateKey,
				URL:           v.URL,
				SSHPort:       v.Gerrit.SSHPort,
			}
		}
		return service.WriteJSON(w, servers, http.StatusOK)
	}
}

func (s *Service) getAllVCSServersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		servers := make(map[string]sdk.VCSConfiguration, len(s.Cfg.Servers))
		for k, v := range s.Cfg.Servers {
			var vcsType string
			if v.Gerrit != nil {
				vcsType = "gerrit"
			} else if v.Bitbucket != nil {
				vcsType = "bitbucket"
			} else if v.Github != nil {
				vcsType = "github"
			} else if v.Gitlab != nil {
				vcsType = "gitlab"
			}

			servers[k] = sdk.VCSConfiguration{
				Type: vcsType,
				URL:  v.URL,
			}
		}
		return service.WriteJSON(w, servers, http.StatusOK)
	}
}

func (s *Service) getVCSServersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		cfg, ok := s.Cfg.Servers[name]
		if !ok {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		s := sdk.VCSConfiguration{
			URL: cfg.URL,
		}
		if cfg.Gerrit != nil {
			s.Type = "gerrit"
		} else if cfg.Bitbucket != nil {
			s.Type = "bitbucket"
		} else if cfg.Github != nil {
			s.Type = "github"
		} else if cfg.Gitlab != nil {
			s.Type = "gitlab"
		}
		return service.WriteJSON(w, s, http.StatusOK)
	}
}

// DEPRECATED VCS
func (s *Service) getVCSServersHooksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {

		// This handler is not called by the 'new VCS'
		// it's managed by GetWebhooksInfos() in API

		// DEPRECATED VCS
		name := muxVar(r, "name")
		cfg, ok := s.Cfg.Servers[name]
		if !ok {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		res := struct {
			WebhooksSupported  bool     `json:"webhooks_supported"`
			WebhooksDisabled   bool     `json:"webhooks_disabled"`
			WebhooksIcon       string   `json:"webhooks_icon"`
			GerritHookDisabled bool     `json:"gerrithook_disabled"`
			Events             []string `json:"events"`
		}{}

		switch {
		case cfg.Bitbucket != nil:
			res.WebhooksSupported = true
			res.WebhooksDisabled = cfg.Bitbucket.DisableWebHooks
			res.WebhooksIcon = sdk.BitbucketIcon
			// https://confluence.atlassian.com/bitbucketserver/event-payload-938025882.html
			res.Events = sdk.BitbucketEvents
		case cfg.BitbucketCloud != nil:
			res.WebhooksSupported = true
			res.WebhooksDisabled = cfg.BitbucketCloud.DisableWebHooks
			res.WebhooksIcon = sdk.BitbucketIcon
			// https://developer.atlassian.com/bitbucket/api/2/reference/resource/hook_events/%7Bsubject_type%7D
			res.Events = sdk.BitbucketCloudEvents
		case cfg.Github != nil:
			res.WebhooksSupported = true
			res.WebhooksDisabled = cfg.Github.DisableWebHooks
			res.WebhooksIcon = sdk.GitHubIcon
			// https://developer.github.com/v3/activity/events/types/
			res.Events = sdk.GitHubEvents
		case cfg.Gitlab != nil:
			res.WebhooksSupported = true
			res.WebhooksDisabled = cfg.Gitlab.DisableWebHooks
			res.WebhooksIcon = sdk.GitlabIcon
			// https://docs.gitlab.com/ee/user/project/integrations/webhooks.html
			res.Events = []string{
				string(gitlab.EventTypePush),
				string(gitlab.EventTypeTagPush),
				string(gitlab.EventTypeIssue),
				string(gitlab.EventTypeNote),
				string(gitlab.EventTypeMergeRequest),
				string(gitlab.EventTypeWikiPage),
				string(gitlab.EventTypePipeline),
				"Job Hook", // TODO update gitlab sdk
			}
		case cfg.Gerrit != nil:
			res.WebhooksSupported = false
			res.GerritHookDisabled = cfg.Gerrit.DisableGerritEvent
			res.WebhooksIcon = sdk.GerritIcon
			// https://git.eclipse.org/r/Documentation/cmd-stream-events.html#events
			res.Events = sdk.GerritEvents
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

// DEPRECATED VCS
func (s *Service) getVCSServersPollingHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		// This handler is not called by the 'new VCS'
		// it's managed by GetPollingInfos() in API

		// DEPRECATED VCS
		name := muxVar(r, "name")
		cfg, ok := s.Cfg.Servers[name]
		if !ok {
			return sdk.WithStack(sdk.ErrNotFound)
		}
		res := struct {
			PollingSupported bool `json:"polling_supported"`
			PollingDisabled  bool `json:"polling_disabled"`
		}{}

		switch {
		case cfg.Bitbucket != nil:
			res.PollingSupported = false
			res.PollingDisabled = cfg.Bitbucket.DisablePolling
		case cfg.BitbucketCloud != nil:
			res.PollingSupported = false
		case cfg.Github != nil:
			res.PollingSupported = true
			res.PollingDisabled = cfg.Github.DisablePolling
		case cfg.Gitlab != nil:
			res.PollingSupported = false
			res.PollingDisabled = cfg.Gitlab.DisablePolling
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func (s *Service) getAuthorizeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		consumer, err := s.getConsumer(name, sdk.VCSAuth{})
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s", name)
		}

		token, url, err := consumer.AuthorizeRedirect(ctx)
		if err != nil {
			return sdk.WrapError(err, "%s", name)
		}

		return service.WriteJSON(w, map[string]string{
			"token": token,
			"url":   url,
		}, http.StatusOK)
	}
}

func (s *Service) postAuthorizeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		consumer, err := s.getConsumer(name, sdk.VCSAuth{})
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable")
		}

		body := map[string]string{}
		if err := service.UnmarshalBody(r, &body); err != nil {
			return err
		}

		token, secret, err := consumer.AuthorizeToken(ctx, body["token"], body["secret"])
		if err != nil {
			return err
		}

		return service.WriteJSON(w, map[string]string{
			"token":  token,
			"secret": secret,
		}, http.StatusOK)
	}
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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

		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		c, err := client.PullRequestCreate(ctx, fmt.Sprintf("%s/%s", owner, repo), prRequest)
		if err != nil {
			return sdk.WrapError(err, "Unable to create pull requests on %s/%s", owner, repo)
		}
		return service.WriteJSON(w, c, http.StatusOK)
	}
}

func (s *Service) postPullRequestCommentHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		var disableStatusDetails bool
		d := r.URL.Query().Get("disableStatusDetails")
		if d != "" {
			disableStatusDetails, _ = strconv.ParseBool(d)
		} else {
			disableStatusDetails = client.IsDisableStatusDetails(ctx)
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
	m := s.NewMonitoringStatus()

	if s.Cfg.Servers["github"].URL != "" {
		m.AddLine(github.GetStatus()...)
	}

	return m
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
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

func (s *Service) postRepoGrantHandler() service.Handler {
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
		// Check if access token has been refreshed
		if vcsAuth.AccessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		if err := client.GrantWritePermission(ctx, owner+"/"+repo); err != nil {
			return sdk.WrapError(err, "unable to grant %s/%s on %s", owner, repo, name)
		}

		return nil
	}
}
