package hooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"
	"gopkg.in/spacemonkeygo/httpsig.v0"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (s *Service) initRouter(ctx context.Context) {
	r := s.Router
	r.Mux.UseEncodedPath()
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.TracingMiddlewareFunc(s))
	r.DefaultAuthMiddleware = service.CheckRequestSignatureMiddleware(s.ParsedAPIPublicKey)
	//r.PostAuthMiddlewares nothing here
	r.PostMiddlewares = append(r.PostMiddlewares, service.TracingPostMiddleware)

	r.Handle("/admin/maintenance", nil, r.POST(s.postMaintenanceHandler))
	r.Handle("/admin/repository/event/{vcsServer}/{repoName}/{uuid}/restart", nil, r.POST(s.postRestartRepositoryHookEventHandler))
	r.Handle("/admin/repository/{vcsServer}/{repoName}", nil, r.DELETE(s.deleteRepositoryHandler))
	r.Handle("/admin/scheduler", nil, r.GET(s.getAllSchedulersHandler))
	r.Handle("/admin/scheduler/{vcsServer}/{repoName}/{workflowName}", nil, r.GET(s.getWorkflowSchedulersHandler))
	r.Handle("/admin/scheduler/execution/{hookID}", nil, r.GET(s.geSchedulerExecutionHandler), r.DELETE(s.deleteSchedulerHandler))
	r.Handle("/admin/outgoing/{projectKey}/{vcsServer}/{repoName}/{workflowName}", nil, r.GET(s.getOutgoingHooksExecutionsByWorkflowHandler))
	r.Handle("/admin/outgoing/{projectKey}/{vcsServer}/{repoName}/{workflowName}/{hookID}", nil, r.GET(s.getOutgoingHookExecutionHandler))

	r.Handle("/mon/version", nil, r.GET(service.VersionHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/status", nil, r.GET(s.statusHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, service.OverrideAuth(service.NoAuthMiddleware)))

	r.Handle("/v2/webhook/repository", nil, r.POST(s.repositoryHooksHandler, service.OverrideAuth(CheckWebhookRequestSignatureMiddleware(s.WebHooksParsedPublicKey))))
	r.Handle("/v2/webhook/repository/{projectKey}/{vcsServerType}/{vcsServer}/{uuid}", nil, r.POST(s.repositoryWebHookHandler, service.OverrideAuth(s.CheckHmac256Signature("X-Hub-Signature-256"))))
	r.Handle("/v2/repository", nil, r.GET(s.listRepositoriesHandler))
	r.Handle("/v2/repository/event/callback", nil, r.POST(s.postRepositoryEventAnalysisCallbackHandler))
	r.Handle("/v2/repository/event/{vcsServer}/{repoName}", nil, r.GET(s.listRepositoryEventHandler), r.DELETE(s.deleteRepositoryEventHandler))
	r.Handle("/v2/repository/event/{vcsServer}/{repoName}/{uuid}", nil, r.GET(s.getRepositoryEventHandler))
	r.Handle("/v2/repository/key/{projectKey}/{vcsServer}/{repoName}", nil, r.POST(s.postGenerateRepositoryWebHookSecretHandler))
	r.Handle("/v2/workflow/manual", nil, r.POST(s.workflowManualHandler))
	r.Handle("/v2/workflow/outgoing", nil, r.POST(s.workflowRunOutgoingEventHandler))

	r.Handle("/v2/workflow/scheduler", nil, r.POST(s.postInstantiateSchedulerHandler))
	r.Handle("/v2/workflow/scheduler/{vcsServer}/{repoName}/{workflowName}", nil, r.DELETE(s.deleteSchedulerByWorkflowHandler))

	r.Handle("/webhook/{uuid}", nil, r.POST(s.webhookHandler, service.OverrideAuth(service.NoAuthMiddleware)), r.GET(s.webhookHandler, service.OverrideAuth(service.NoAuthMiddleware)), r.DELETE(s.webhookHandler, service.OverrideAuth(service.NoAuthMiddleware)), r.PUT(s.webhookHandler, service.OverrideAuth(service.NoAuthMiddleware)))
	r.Handle("/task", nil, r.POST(s.postTaskHandler), r.GET(s.getTasksHandler))
	r.Handle("/task/bulk/start", nil, r.GET(s.startTasksHandler))
	r.Handle("/task/bulk/stop", nil, r.GET(s.stopTasksHandler))
	r.Handle("/task/bulk", nil, r.POST(s.postTaskBulkHandler), r.DELETE(s.deleteTaskBulkHandler))
	r.Handle("/task/execute", nil, r.POST(s.postAndExecuteTaskHandler))
	r.Handle("/task/{uuid}", nil, r.GET(s.getTaskHandler), r.PUT(s.putTaskHandler), r.DELETE(s.deleteTaskHandler))
	r.Handle("/task/{uuid}/start", nil, r.GET(s.startTaskHandler))
	r.Handle("/task/{uuid}/stop", nil, r.GET(s.stopTaskHandler))
	r.Handle("/task/{uuid}/execution", nil, r.GET(s.getTaskExecutionsHandler), r.DELETE(s.deleteAllTaskExecutionsHandler))
	r.Handle("/task/{uuid}/execution/{timestamp}", nil, r.GET(s.getTaskExecutionHandler))
	r.Handle("/task/{uuid}/execution/{timestamp}/stop", nil, r.POST(s.postStopTaskExecutionHandler))
}

func (s *Service) CheckHmac256Signature(headerName string) service.Middleware {
	return func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
		signHeaderValue := req.Header.Get(headerName)
		if signHeaderValue == "" {
			return ctx, sdk.NewErrorFrom(sdk.ErrUnauthorized, "unable to check signature")
		}
		vars := mux.Vars(req)
		projectKey := vars["projectKey"]
		vcsType := vars["vcsServerType"]
		vcsName := vars["vcsServer"]
		uuid := vars["uuid"]

		defer req.Body.Close() // nolint
		body, err := io.ReadAll(req.Body)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return ctx, sdk.NewErrorFrom(sdk.ErrUnauthorized, "unable to check signature")
		}

		eventName, err := s.extractEventFromHeader(ctx, vcsType, req.Header)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return ctx, sdk.NewErrorFrom(sdk.ErrUnauthorized, "unable to extract event from header")
		}

		repoName, _, err := s.extractDataFromPayload(req.Header, vcsType, body, eventName)
		if err != nil {
			log.ErrorWithStackTrace(ctx, err)
			return ctx, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find repository")
		}

		newRequestBody := io.NopCloser(bytes.NewBuffer(body))
		req.Body = newRequestBody

		// Create a new HMAC by defining the hash type and the key (as byte array)
		hookKey := sdk.GenerateRepositoryWebHookSecret(s.Cfg.RepositoryWebHookKey, projectKey, vcsName, repoName, uuid)
		h := hmac.New(sha256.New, []byte(hookKey))
		h.Write(body)
		sha := hex.EncodeToString(h.Sum(nil))
		if strings.TrimPrefix(signHeaderValue, "sha256=") != sha {
			log.Error(ctx, "signature mismatch: got %s, compute %s", signHeaderValue, sha)
			return ctx, sdk.NewErrorFrom(sdk.ErrUnauthorized, "wrong signature")
		}

		// Check uuid
		if _, err := s.Client.ProjectWebHookGet(ctx, projectKey, uuid); err != nil {
			log.Error(ctx, "unable to retrieve hook %s/%s: %v", projectKey, uuid, err)
			return ctx, sdk.NewErrorFrom(sdk.ErrUnauthorized, "hook expired")
		}

		return ctx, nil
	}
}

type webhookHttpVerifier struct {
	sync.Mutex
	pubKey *rsa.PublicKey
}

func (v *webhookHttpVerifier) SetKey(pubKey *rsa.PublicKey) {
	v.Lock()
	defer v.Unlock()
	v.pubKey = pubKey
}

func (v *webhookHttpVerifier) GetKey(_ string) interface{} {
	v.Lock()
	defer v.Unlock()
	return v.pubKey
}

var (
	webhookHTTPVerifier *webhookHttpVerifier
)

func CheckWebhookRequestSignatureMiddleware(pubKey *rsa.PublicKey) service.Middleware {
	webhookHTTPVerifier = new(webhookHttpVerifier)
	webhookHTTPVerifier.SetKey(pubKey)

	verifier := httpsig.NewVerifier(webhookHTTPVerifier)
	verifier.SetRequiredHeaders([]string{"(request-target)", "host", "date", sdk.SignHeaderVCSType, sdk.SignHeaderVCSName, sdk.SignHeaderRepoName, sdk.SignHeaderEventName})

	return func(ctx context.Context, w http.ResponseWriter, req *http.Request, rc *service.HandlerConfig) (context.Context, error) {
		if err := verifier.Verify(req); err != nil {
			return ctx, sdk.NewError(sdk.ErrUnauthorized, err)
		}

		log.Debug(ctx, "Request has been successfully verified")
		return ctx, nil
	}
}
