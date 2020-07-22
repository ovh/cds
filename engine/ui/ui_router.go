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
	r.SetHeaderFunc = api.DefaultHeaders
	r.PostMiddlewares = append(r.PostMiddlewares, api.TracingPostMiddleware)

	serviceURL, _ := url.Parse(s.Cfg.URL)

	r.Handle(serviceURL.Path+"/mon/version", nil, r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle(serviceURL.Path+"/mon/status", nil, r.GET(s.statusHandler, api.Auth(false)))
	r.Handle(serviceURL.Path+"/mon/metrics", nil, r.GET(service.GetPrometheustMetricsHandler(s), api.Auth(false)))
	r.Handle(serviceURL.Path+"/mon/metrics/all", nil, r.GET(service.GetMetricsHandler, api.Auth(false)))

	// proxypass
	r.Mux.PathPrefix(serviceURL.Path + "/cdsapi").Handler(s.getReverseProxy(serviceURL.Path+"/cdsapi", s.Cfg.API.HTTP.URL))
	r.Mux.PathPrefix(serviceURL.Path + "/cdshooks").Handler(s.getReverseProxy(serviceURL.Path+"/cdshooks", s.Cfg.HooksURL))

	// serve static UI files
	r.Mux.PathPrefix("/").Handler(s.uiServe(http.Dir(s.HTMLDir)))
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

func (s *Service) uiServe(fs http.FileSystem) http.Handler {
	fsh := http.FileServer(fs)
	serviceURL, _ := url.Parse(s.Cfg.URL)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimPrefix(r.URL.Path, serviceURL.Path)
		filePath := path.Clean(r.URL.Path)
		_, err := fs.Open(filePath)
		if os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(s.HTMLDir, "index.html"))
			return
		}
		fsh.ServeHTTP(w, r)
	})
}
