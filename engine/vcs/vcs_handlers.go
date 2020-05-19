package vcs

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/xanzy/go-gitlab"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/engine/vcs/github"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func muxVar(r *http.Request, s string) string {
	vars := mux.Vars(r)
	return vars[s]
}

// QueryString return a string from a query parameter
func QueryString(r *http.Request, s string) string {
	return r.FormValue(s)
}

func (s *Service) getAllVCSServersHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		servers := make(map[string]sdk.VCSConfiguration, len(s.Cfg.Servers))
		for k, v := range s.Cfg.Servers {
			var vcsType, user, privateKey string
			var sshPort int
			if v.Gerrit != nil {
				vcsType = "gerrit"
				user = v.Gerrit.EventStream.User
				privateKey = v.Gerrit.EventStream.PrivateKey
				sshPort = v.Gerrit.SSHPort
			} else if v.Bitbucket != nil {
				vcsType = "bitbucket"
			} else if v.Github != nil {
				vcsType = "github"
			} else if v.Gitlab != nil {
				vcsType = "gitlab"
			}

			servers[k] = sdk.VCSConfiguration{
				Type:     vcsType,
				Username: user,
				Password: privateKey,
				URL:      v.URL,
				SSHPort:  sshPort,
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

func (s *Service) getVCSServersHooksHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
			res.Events = []string{
				"repo:refs_changed",
				"repo:modified",
				"repo:forked",
				"repo:comment:added",
				"repo:comment:edited",
				"repo:comment:deleted",
				"pr:opened",
				"pr:modified",
				"pr:reviewer:updated",
				"pr:reviewer:approved",
				"pr:reviewer:unapproved",
				"pr:reviewer:needs_work",
				"pr:merged",
				"pr:declined",
				"pr:deleted",
				"pr:comment:added",
				"pr:comment:edited",
				"pr:comment:deleted",
			}
		case cfg.BitbucketCloud != nil:
			res.WebhooksSupported = true
			res.WebhooksDisabled = cfg.BitbucketCloud.DisableWebHooks
			res.WebhooksIcon = sdk.BitbucketIcon
			// https://developer.atlassian.com/bitbucket/api/2/reference/resource/hook_events/%7Bsubject_type%7D
			res.Events = []string{
				"repo:push",
				"pullrequest:unapproved",
				"issue:comment_created",
				"pullrequest:approved",
				"repo:created",
				"repo:deleted",
				"repo:imported",
				"pullrequest:comment_updated",
				"issue:updated",
				"project:updated",
				"pullrequest:comment_created",
				"repo:commit_status_updated",
				"pullrequest:updated",
				"issue:created",
				"repo:fork",
				"pullrequest:comment_deleted",
				"repo:commit_status_created",
				"repo:updated",
				"pullrequest:rejected",
				"pullrequest:fulfilled",
				"pullrequest:created",
				"repo:transfer",
				"repo:commit_comment_created",
			}
		case cfg.Github != nil:
			res.WebhooksSupported = true
			res.WebhooksDisabled = cfg.Github.DisableWebHooks
			res.WebhooksIcon = sdk.GitHubIcon
			// https://developer.github.com/v3/activity/events/types/
			res.Events = []string{
				"push",
				"check_run",
				"check_suite",
				"commit_comment",
				"create",
				"delete",
				"deployment",
				"deployment_status",
				"fork",
				"github_app_authorization",
				"gollum",
				"installation",
				"installation_repositories",
				"issue_comment",
				"issues",
				"label",
				"marketplace_purchase",
				"member",
				"membership",
				"milestone",
				"organization",
				"org_block",
				"page_build",
				"project_card",
				"project_column",
				"project",
				"public",
				"pull-request_review_comment",
				"pull-request_review",
				"pull_request",
				"repository",
				"repository_import",
				"repository_vulnerability_alert",
				"release",
				"security_advisory",
				"status",
				"team",
				"team_add",
				"watch",
			}
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
			res.Events = []string{
				"patchset-created",
				"assignee-changed",
				"change-abandoned",
				"change-deleted",
				"change-merged",
				"change-restored",
				"comment-added",
				"draft-published",
				"dropped-output",
				"hashtags-changed",
				"project-created",
				"ref-updated",
				"reviewer-added",
				"reviewer-deleted",
				"topic-changed",
				"wip-state-changed",
				"private-state-changed",
				"vote-deleted",
			}
		}

		return service.WriteJSON(w, res, http.StatusOK)
	}
}

func (s *Service) getVCSServersPollingHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
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
		consumer, err := s.getConsumer(name)
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

func (s *Service) postAuhorizeHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		consumer, err := s.getConsumer(name)
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getReposHandler> Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client")
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getRepoHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getBranchesHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		branches, err := client.Branches(ctx, fmt.Sprintf("%s/%s", owner, repo))
		if err != nil {
			return sdk.WrapError(err, "Unable to get repo %s/%s branches", owner, repo)
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getBranchHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		ghBranch, err := client.Branch(ctx, fmt.Sprintf("%s/%s", owner, repo), branch)
		if err != nil {
			return sdk.WrapError(err, "Unable to get repo %s/%s branch %s", owner, repo, branch)
		}
		return service.WriteJSON(w, ghBranch, http.StatusOK)
	}
}

func (s *Service) getTagsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		log.Debug("getTagsHandler>")

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getTagsHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		log.Debug("getCommitsHandler>")

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getCommitsHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		log.Debug("getCommitsBetweenRefsHandler>")

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getCommitsBetweenRefsHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getCommitHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getCommitStatusHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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
		sid := muxVar(r, "id")
		id, err := strconv.Atoi(sid)
		if err != nil {
			return sdk.WithStack(sdk.ErrWrongRequest)
		}

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getPullRequestsHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getPullRequestsHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getEventsHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "Unable to get access token headers")
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable")
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client")
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		if err := client.SetStatus(ctx, evt); err != nil {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> postReleaseHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		if err := client.UploadReleaseFile(ctx, fmt.Sprintf("%s/%s", owner, repo), release, uploadURL, artifactName, r.Body); err != nil {
			return sdk.WrapError(err, "Unable to upload release file %s %s/%s", name, owner, repo)
		}

		return nil
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		body := sdk.VCSHook{}
		if err := service.UnmarshalBody(r, &body); err != nil {
			return sdk.WrapError(err, "Unable to read body %s %s/%s", name, owner, repo)
		}

		if err := client.CreateHook(ctx, fmt.Sprintf("%s/%s", owner, repo), &body); err != nil {
			return sdk.WrapError(err, "CreateHook %s %s/%s", name, owner, repo)
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

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> deleteHookHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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
				ID:       hookID,
				Workflow: true,
			}
		}

		return client.DeleteHook(ctx, fmt.Sprintf("%s/%s", owner, repo), hook)
	}
}

func (s *Service) getListForks() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> getListForks> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
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
func (s *Service) Status(ctx context.Context) sdk.MonitoringStatus {
	m := s.CommonMonitoring()

	if s.Cfg.Servers["github"].URL != "" {
		m.Lines = append(m.Lines, github.GetStatus()...)
	}

	return m
}

func (s *Service) statusHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var status = http.StatusOK
		return service.WriteJSON(w, s.Status(ctx), status)
	}
}

func (s *Service) postRepoGrantHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		name := muxVar(r, "name")
		owner := muxVar(r, "owner")
		repo := muxVar(r, "repo")

		accessToken, accessTokenSecret, created, ok := getAccessTokens(ctx)
		if !ok {
			return sdk.WrapError(sdk.ErrUnauthorized, "VCS> postRepoGrantHandler> Unable to get access token headers %s %s/%s", name, owner, repo)
		}

		consumer, err := s.getConsumer(name)
		if err != nil {
			return sdk.WrapError(err, "VCS server unavailable %s %s/%s", name, owner, repo)
		}

		client, err := consumer.GetAuthorizedClient(ctx, accessToken, accessTokenSecret, created)
		if err != nil {
			return sdk.WrapError(err, "Unable to get authorized client %s %s/%s", name, owner, repo)
		}
		// Check if access token has been refreshed
		if accessToken != client.GetAccessToken(ctx) {
			w.Header().Set(sdk.HeaderXAccessToken, client.GetAccessToken(ctx))
		}

		if err := client.GrantWritePermission(ctx, owner+"/"+repo); err != nil {
			return sdk.WrapError(err, "unable to grant %s/%s on %s", owner, repo, name)
		}

		return nil
	}
}
