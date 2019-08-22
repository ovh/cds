package ui

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/ovh/cds/engine/api"
	"github.com/ovh/cds/sdk/log"
)

func (s *Service) initRouter(ctx context.Context) {
	log.Debug("UI> Router initialized")
	r := s.Router
	r.Background = ctx
	r.URL = s.Cfg.URL
	r.SetHeaderFunc = api.DefaultHeaders
	r.PostMiddlewares = append(r.PostMiddlewares, api.TracingPostMiddleware)

	r.Handle("/mon/version", r.GET(api.VersionHandler, api.Auth(false)))
	r.Handle("/mon/status", r.GET(s.statusHandler, api.Auth(false)))

	// proxypass
	r.Mux.PathPrefix("/cdsapi").Handler(s.getReverseProxy("/cdsapi", s.Cfg.API.HTTP.URL))
	r.Mux.PathPrefix("/cdshooks").Handler(s.getReverseProxy("/cdshooks", s.Cfg.HooksURL))

	// serve static UI files
	r.Mux.PathPrefix("/").Handler(s.uiServe(http.Dir(s.HTMLDir)))
}

func (s *Service) getReverseProxy(path, urlRemote string) *httputil.ReverseProxy {
	origin, _ := url.Parse(urlRemote)
	director := func(req *http.Request) {
		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Origin-Host", origin.Host)
		req.URL.Scheme = "http"
		req.URL.Host = origin.Host
		req.URL.Path = strings.TrimPrefix(req.URL.Path, path)
	}
	return &httputil.ReverseProxy{Director: director}
}

func (s *Service) uiServe(fs http.FileSystem) http.Handler {
	fsh := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fs.Open(path.Clean(r.URL.Path))
		if os.IsNotExist(err) {
			http.ServeFile(w, r, s.HTMLDir+"/index.html")
			return
		}
		fsh.ServeHTTP(w, r)
	})
}
