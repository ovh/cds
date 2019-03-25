package api

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) downloadsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		resources := sdk.AllDownloadableResourcesWithAvailability(api.Config.Directories.Download)
		return service.WriteJSON(w, resources, http.StatusAccepted)
	}
}

func (api *API) downloadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		os := vars["os"]
		arch := vars["arch"]
		r.ParseForm() // nolint
		variant := r.Form.Get("variant")

		filename := sdk.GetArtifactFilename(name, os, arch, variant)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, filename))

		path := path.Join(api.Config.Directories.Download, filename)
		log.Debug("downloading from %s", path)

		http.ServeFile(w, r, path)
		return nil
	}
}
