package vcs

import (
	"context"

	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/service"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug(ctx, "VCS> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.TracingMiddlewareFunc(s))
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey)
	r.PostAuthMiddlewares = append(r.PostAuthMiddlewares, s.authMiddleware)
	r.PostMiddlewares = append(r.PostMiddlewares, service.TracingPostMiddleware)

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/vcs", nil, r.GET(s.getAllVCSServersHandler))
	r.Handle("/vcsgerrit", nil, r.GET(s.getVCSGerritHandler))
	r.Handle("/vcs/{name}", nil, r.GET(s.getVCSServersHandler))
	r.Handle("/vcs/{name}/webhooks", nil, r.GET(s.getVCSServersHooksHandler))
	r.Handle("/vcs/{name}/polling", nil, r.GET(s.getVCSServersPollingHandler))

	r.Handle("/vcs/{name}/authorize", nil, r.GET(s.getAuthorizeHandler), r.POST(s.postAuthorizeHandler))

	r.Handle("/vcs/{name}/repos", nil, r.GET(s.getReposHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}", nil, r.GET(s.getRepoHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches", nil, r.GET(s.getBranchesHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches/", nil, r.GET(s.getBranchHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches/commits", nil, r.GET(s.getCommitsHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/tags", nil, r.GET(s.getTagsHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits", nil, r.GET(s.getCommitsBetweenRefsHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits/{commit}", nil, r.GET(s.getCommitHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits/{commit}/statuses", nil, r.GET(s.getCommitStatusHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/grant", nil, r.POST(s.postRepoGrantHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests", nil, r.GET(s.getPullRequestsHandler), r.POST(s.postPullRequestsHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests/comments", nil, r.POST(s.postPullRequestCommentHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests/{id}", nil, r.GET(s.getPullRequestHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/events", nil, r.GET(s.getEventsHandler), r.POST(s.postFilterEventsHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/hooks", nil, r.GET(s.getHookHandler), r.POST(s.postHookHandler), r.PUT(s.putHookHandler), r.DELETE(s.deleteHookHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/releases", nil, r.POST(s.postReleaseHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/releases/{release}/artifacts/{artifactName}", nil, r.POST(s.postUploadReleaseFileHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/forks", nil, r.GET(s.getListForks))

	r.Handle("/vcs/{name}/status", nil, r.POST(s.postStatusHandler))
}
