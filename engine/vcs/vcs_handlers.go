package vcs

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/sdk"

	"github.com/ovh/cds/engine/api"
)

func (s *Service) getAllVCSServersHandler() api.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return api.WriteJSON(w, r, s.Cfg.Servers, http.StatusOK)
	}
}

func muxVar(r *http.Request, s string) string {
	vars := mux.Vars(r)
	return vars[s]
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
		branch := muxVar(r, "branch")

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
		branch := muxVar(r, "branch")
		since := r.URL.Query().Get("since")
		until := r.URL.Query().Get("until")

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
