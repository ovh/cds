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
	r.Middlewares = append(r.Middlewares, s.authMiddleware)

	r.Handle("/mon/version", r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", r.GET(s.statusHandler, api.Auth(false)))

	r.Handle("/vcs", r.GET(s.getAllVCSServersHandler))
	r.Handle("/vcs/{name}", r.GET(s.getVCSServersHandler))
	r.Handle("/vcs/{name}/webhooks", r.GET(s.getVCSServersHooksHandler))
	r.Handle("/vcs/{name}/polling", r.GET(s.getVCSServersPollingHandler))

	r.Handle("/vcs/{name}/authorize", r.GET(s.getAuthorizeHandler), r.POST(s.postAuhorizeHandler))

	r.Handle("/vcs/{name}/repos", r.GET(s.getReposHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}", r.GET(s.getRepoHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches", r.GET(s.getBranchesHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches/", r.GET(s.getBranchHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/branches/commits", r.GET(s.getCommitsHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits/{commit}", r.GET(s.getCommitHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/commits/{commit}/statuses", r.GET(s.getCommitStatusHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/grant", r.POST(s.postRepoGrantHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests", r.GET(s.getPullRequestsHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/pullrequests/{id}/comments", r.POST(s.postPullRequestCommentHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/events", r.GET(s.getEventsHandler), r.POST(s.postFilterEventsHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/hooks", r.GET(s.getHookHandler), r.POST(s.postHookHandler), r.DELETE(s.deleteHookHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/releases", r.POST(s.postReleaseHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/releases/{release}/artifacts/{artifactName}", r.POST(s.postUploadReleaseFileHandler))
	r.Handle("/vcs/{name}/repos/{owner}/{repo}/forks", r.GET(s.getListForks))

	r.Handle("/vcs/{name}/status", r.POST(s.postStatusHandler))
}
