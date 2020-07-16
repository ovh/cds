package vcs

import (
	"context"

	"github.com/ovh/cds/engine/service"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug("VCS> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey), s.authMiddleware, api.TracingMiddlewareFunc(s, nil, nil))
	r.PostMiddlewares = append(r.PostMiddlewares, api.TracingPostMiddleware)

	r.Handle("/mon/version", nil, r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, api.Auth(false)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), api.Auth(false)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, api.Auth(false)))

	r.Handle("/vcs", nil, r.GET(s.getAllVCSServersHandler))
	r.Handle("/vcs/{name}", nil, r.GET(s.getVCSServersHandler))
	r.Handle("/vcs/{name}/webhooks", nil, r.GET(s.getVCSServersHooksHandler))
	r.Handle("/vcs/{name}/polling", nil, r.GET(s.getVCSServersPollingHandler))

	r.Handle("/vcs/{name}/authorize", nil, r.GET(s.getAuthorizeHandler), r.POST(s.postAuhorizeHandler))

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
