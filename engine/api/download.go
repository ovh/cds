package api

import (
	"context"
	"fmt"
	"net/http"
	"path"

	"github.com/gorilla/mux"

	"github.com/ovh/cds/sdk"
)

func (api *API) downloadsHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return WriteJSON(w, r, sdk.GetStaticDownloads(), http.StatusAccepted)
	}
}

func (api *API) downloadHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		os := vars["os"]

		arch, err := sdk.IsBinaryOSArchValid(name, os, vars["arch"])
		if err != nil {
			return err
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, name))

		path := path.Join(api.Config.Directories.Download, fmt.Sprintf("cds-%s-%s-%s", name, os, arch))

		http.ServeFile(w, r, path)
		return nil
	}
}
