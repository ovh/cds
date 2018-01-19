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
		downloads := sdk.GetStaticDownloadsWithAvailability(api.Config.Directories.Download)
		return WriteJSON(w, r, downloads, http.StatusAccepted)
	}
}

func (api *API) downloadHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		name := vars["name"]
		os := vars["os"]

		arch, extension, err := sdk.IsBinaryOSArchValid(api.Config.Directories.Download, name, os, vars["arch"])
		if err != nil {
			return err
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s%s"`, name, extension))

		path := path.Join(api.Config.Directories.Download, fmt.Sprintf("%s%s-%s-%s%s", sdk.DownloadGetPrefix(name), name, os, arch, extension))

		http.ServeFile(w, r, path)
		return nil
	}
}
