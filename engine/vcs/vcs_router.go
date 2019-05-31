package vcs

import (
	"context"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug("VCS> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders
	r.Middlewares = append(r.Middlewares, s.authMiddleware, api.TracingMiddlewareFunc(s.ServiceName, nil, nil))
	r.PostMiddlewares = append(r.PostMiddlewares, api.TracingPostMiddleware)

	r.Handle("/mon/version", nil, r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, api.Auth(false)))

	r.Handle("/vcs", nil, r.GET(s.getAllVCSServersHandler))
	r.Handle("/vcs/{name}", nil, r.GET(s.getVCSServersHandler))
	r.Handle("/vcs/{name}/webhooks", nil, r.GET(s.getVCSServersHooksHandler))
	r.Handle("/vcs/{name}/polling", nil, r.GET(s.getVCSServersPollingHandler))

	r.Handle("/vcs/{name}/authorize", nil, r.GET(s.getAuthorizeHandler), r.POST(s.postAuhorizeHandler))

	r.Handle("/vcs/{name}/repos", nil, r.GET(s.getReposHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}", nil, r.GET(s.getRepoHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches", nil, r.GET(s.getBranchesHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches/", nil, r.GET(s.getBranchHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches/commits", nil, r.GET(s.getCommitsHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/tags", nil, r.GET(s.getTagsHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits", nil, r.GET(s.getCommitsBetweenRefsHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits/{commit}", nil, r.GET(s.getCommitHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits/{commit}/statuses", nil, r.GET(s.getCommitStatusHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/grant", nil, r.POST(s.postRepoGrantHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests", nil, r.GET(s.getPullRequestsHandler, api.EnableTracing()), r.POST(s.postPullRequestsHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests/{id}", nil, r.GET(s.getPullRequestHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests/{id}/comments", nil, r.POST(s.postPullRequestCommentHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/events", nil, r.GET(s.getEventsHandler, api.EnableTracing()), r.POST(s.postFilterEventsHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/hooks", nil, r.GET(s.getHookHandler, api.EnableTracing()), r.POST(s.postHookHandler, api.EnableTracing()), r.DELETE(s.deleteHookHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/releases", nil, r.POST(s.postReleaseHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/releases/{release}/artifacts/{artifactName}", nil, r.POST(s.postUploadReleaseFileHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/forks", nil, r.GET(s.getListForks, api.EnableTracing()))

	r.Handle("/vcs/{name}/status", nil, r.POST(s.postStatusHandler, api.EnableTracing()))
}
