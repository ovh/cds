package ui

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ovh/cds/engine/service"
	"github.com/rockbears/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug(ctx, "ui> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.Middlewares = append(r.Middlewares, service.TracingMiddlewareFunc(s))
	r.DefaultAuthMiddleware = service.NoAuthMiddleware
	r.PostMiddlewares = append(r.PostMiddlewares, service.TracingPostMiddleware)

	r.Handle(s.Cfg.DeployURL+"/mon/version", nil, r.GET(service.VersionHandler))
	r.Handle(s.Cfg.DeployURL+"/mon/status", nil, r.GET(s.statusHandler))
	r.Handle(s.Cfg.DeployURL+"/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s)))
	r.Handle(s.Cfg.DeployURL+"/mon/metrics/all", nil, r.GET(service.GetMetricsHandler))

	// proxypass if enabled
	if s.Cfg.EnableServiceProxy {
		if s.Cfg.API.HTTP.URL != "" {
			apiPath := s.Cfg.DeployURL + "/cdsapi"
			r.Mux.PathPrefix(apiPath).Handler(s.getReverseProxy(ctx, apiPath, s.Cfg.API.HTTP.URL))
		}
		if s.Cfg.HooksURL != "" {
			hooksPath := s.Cfg.DeployURL + "/cdshooks"
			r.Mux.PathPrefix(hooksPath).Handler(s.getReverseProxy(ctx, hooksPath, s.Cfg.HooksURL))
		}
		if s.Cfg.CDNURL != "" {
			cdnPath := s.Cfg.DeployURL + "/cdscdn"
			r.Mux.PathPrefix(cdnPath).Handler(s.getReverseProxy(ctx, cdnPath, s.Cfg.CDNURL))
		}
	}

	// serve static UI files
	r.Mux.PathPrefix("/docs").Handler(s.uiServe(http.Dir(s.DocsDir), s.DocsDir))
	r.Mux.PathPrefix("/").Handler(s.uiServe(http.Dir(s.HTMLDir), s.HTMLDir))
}

func (s *Service) getReverseProxy(ctx context.Context, path, urlRemote string) http.Handler {
	filter := func(req *http.Request) bool {
		reqPath := strings.TrimPrefix(req.URL.Path, path)

		// on api proxypass, deny request on /mon/metrics
		if strings.HasSuffix(path, "api") && strings.HasPrefix(reqPath, "/mon/metrics") {
			return false
		}

		// on hooks proxypass, allow only request on /webhook/ path
		if strings.HasSuffix(path, "hooks") && !strings.HasPrefix(reqPath, "/webhook/") {
			return false
		}

		// on cdn proxypass, allow only request on /item
		if strings.HasSuffix(path, "cdn") && !strings.HasPrefix(reqPath, "/item") {
			return false
		}

		return true
	}

	origin, _ := url.Parse(urlRemote)
	director := func(req *http.Request) {
		reqPath := strings.TrimPrefix(req.URL.Path, path)

		var clientIP string
		if s.Cfg.HTTP.HeaderXForwardedFor != "" {
			// Retrieve the client ip address from the header (X-Forwarded-For by default)
			clientIP = req.Header.Get(s.Cfg.HTTP.HeaderXForwardedFor)
		}
		if clientIP == "" {
			// If the header has not been found, fallback on the remote address from the http request
			clientIP = req.RemoteAddr
		}

		headerForward := "X-Forwarded-For"
		if s.Cfg.HTTP.HeaderXForwardedFor != "" {
			headerForward = s.Cfg.HTTP.HeaderXForwardedFor
		}

		req.Header.Add(headerForward, clientIP)
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Origin-Host", origin.Host)
		req.URL.Scheme = origin.Scheme
		req.URL.Host = origin.Host
		req.URL.Path = origin.Path + reqPath
		req.Host = origin.Host
	}

	return &reverseProxyWithFilter{
		ctx:    ctx,
		rp:     &httputil.ReverseProxy{Director: director},
		filter: filter,
	}
}

type reverseProxyWithFilter struct {
	ctx    context.Context
	rp     *httputil.ReverseProxy
	filter func(r *http.Request) bool
}

func (r *reverseProxyWithFilter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !r.filter(req) {
		log.Debug(r.ctx, "proxy deny on target route")
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	r.rp.ServeHTTP(rw, req)
}

func (s *Service) uiServe(fs http.FileSystem, dir string) http.Handler {
	fsh := http.FileServer(uiFileSystem{fs})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dir == s.DocsDir {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/docs")
		} else {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, s.Cfg.DeployURL)
		}
		filePath := path.Clean(r.URL.Path)
		_, err := fs.Open(filePath)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fsh.ServeHTTP(w, r)
	})
}

type uiFileSystem struct {
	base http.FileSystem
}

func (uifs uiFileSystem) Open(path string) (http.File, error) {
	f, err := uifs.base.Open(path)
	if err != nil {
		return nil, os.ErrNotExist
	}

	s, err := f.Stat()
	if err != nil {
		return nil, os.ErrNotExist
	}
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := uifs.base.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}

			return nil, os.ErrNotExist
		}
	}

	return f, nil
}
