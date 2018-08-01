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

	r.Handle("/mon/version", r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", r.GET(s.statusHandler, api.Auth(false)))

	r.Handle("/vcs", r.GET(s.getAllVCSServersHandler))
	r.Handle("/vcs/{name}", r.GET(s.getVCSServersHandler))
	r.Handle("/vcs/{name}/webhooks", r.GET(s.getVCSServersHooksHandler))
	r.Handle("/vcs/{name}/polling", r.GET(s.getVCSServersPollingHandler))

	r.Handle("/vcs/{name}/authorize", r.GET(s.getAuthorizeHandler), r.POST(s.postAuhorizeHandler))

	r.Handle("/vcs/{name}/repos", r.GET(s.getReposHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}", r.GET(s.getRepoHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches", r.GET(s.getBranchesHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches/", r.GET(s.getBranchHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches/commits", r.GET(s.getCommitsHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits/{commit}", r.GET(s.getCommitHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits/{commit}/statuses", r.GET(s.getCommitStatusHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/grant", r.POST(s.postRepoGrantHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests", r.GET(s.getPullRequestsHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests/{id}/comments", r.POST(s.postPullRequestCommentHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/events", r.GET(s.getEventsHandler, api.EnableTracing()), r.POST(s.postFilterEventsHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/hooks", r.GET(s.getHookHandler, api.EnableTracing()), r.POST(s.postHookHandler, api.EnableTracing()), r.DELETE(s.deleteHookHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/releases", r.POST(s.postReleaseHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/releases/{release}/artifacts/{artifactName}", r.POST(s.postUploadReleaseFileHandler, api.EnableTracing()))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/forks", r.GET(s.getListForks, api.EnableTracing()))

	r.Handle("/vcs/{name}/status", r.POST(s.postStatusHandler, api.EnableTracing()))
}
