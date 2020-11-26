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

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug("ui> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = service.DefaultHeaders
	r.DefaultAuthMiddleware = service.NoAuthMiddleware
	r.PostMiddlewares = append(r.PostMiddlewares, api.TracingPostMiddleware)

	r.Handle(s.Cfg.DeployURL+"/mon/version", nil, r.GET(service.VersionHandler))
	r.Handle(s.Cfg.DeployURL+"/mon/status", nil, r.GET(s.statusHandler))
	r.Handle(s.Cfg.DeployURL+"/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s)))
	r.Handle(s.Cfg.DeployURL+"/mon/metrics/all", nil, r.GET(service.GetMetricsHandler))

	// proxypass
	r.Mux.PathPrefix(s.Cfg.DeployURL + "/cdsapi").Handler(s.getReverseProxy(s.Cfg.DeployURL+"/cdsapi", s.Cfg.API.HTTP.URL))
	r.Mux.PathPrefix(s.Cfg.DeployURL + "/cdshooks").Handler(s.getReverseProxy(s.Cfg.DeployURL+"/cdshooks", s.Cfg.HooksURL))
	r.Mux.PathPrefix(s.Cfg.DeployURL + "/cdscdn").Handler(s.getReverseProxy(s.Cfg.DeployURL+"/cdscdn", s.Cfg.CDNURL))

	// serve static UI files
	r.Mux.PathPrefix("/docs").Handler(s.uiServe(http.Dir(s.DocsDir), s.DocsDir))
	r.Mux.PathPrefix("/").Handler(s.uiServe(http.Dir(s.HTMLDir), s.HTMLDir))
}

func (s *Service) getReverseProxy(path, urlRemote string) *httputil.ReverseProxy {
	origin, _ := url.Parse(urlRemote)

	director := func(req *http.Request) {
		reqPath := strings.TrimPrefix(req.URL.Path, path)
		// on proxypass /cdshooks, allow only request on /webhook/ path
		if strings.HasSuffix(path, "/cdshooks") && !strings.HasPrefix(reqPath, "/webhook/") {
			// return 502 bad gateway
			req = &http.Request{} // nolint
		} else {
			req.Header.Add("X-Forwarded-Host", req.Host)
			req.Header.Add("X-Origin-Host", origin.Host)
			req.URL.Scheme = origin.Scheme
			req.URL.Host = origin.Host
			req.URL.Path = origin.Path + reqPath
			req.Host = origin.Host
		}
	}
	return &httputil.ReverseProxy{Director: director}
}

func (s *Service) uiServe(fs http.FileSystem, dir string) http.Handler {
	fsh := http.FileServer(fs)
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
