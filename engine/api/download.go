package api

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/download"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

func (api *API) downloadsHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		resources := sdk.AllDownloadableResources()
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

		if err := download.CheckBinary(ctx, api.getDownloadConf(), name, os, arch, variant); err != nil {
			return err
		}

		filename := sdk.BinaryFilename(name, os, arch, variant)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, filename))

		path := path.Join(api.Config.Download.Directory, filename)
		log.Debug(ctx, "downloading from %s", path)

		http.ServeFile(w, r, path)
		return nil
	}
}
