package api

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"strings"

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
		fragment := strings.Split(strings.TrimPrefix(r.URL.Path, "/download/"), "/")

		var name, os, arch string
		if len(fragment) == 1 {
			if !sdk.Assets.Contains(fragment[0]) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given file name: %q", fragment[0])
			}
		} else if len(fragment) == 3 {
			if !sdk.Binaries.Contains(fragment[0]) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given binary name: %q", fragment[0])
			}
			if !sdk.SupportedOS.Contains(fragment[1]) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given OS: %q", fragment[1])
			}
			if !sdk.SupportedARCH.Contains(sdk.GetArchName(fragment[2])) {
				return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid given ARCH: %q", fragment[2])
			}
		} else {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "given file path should be like binary-name/os/arch")
		}

		name = sdk.NoPath(fragment[0])
		if len(fragment) == 3 {
			os = sdk.NoPath(fragment[1])
			arch = sdk.NoPath(fragment[2])
		}

		r.ParseForm() // nolint
		variant := sdk.NoPath(r.Form.Get("variant"))
		if !sdk.IsInArray(variant, []string{"", "nokeychain", "keychain"}) {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "invalid variant: %s", variant)
		}

		if err := download.CheckBinary(ctx, api.getDownloadConf(), name, os, arch, variant); err != nil {
			return err
		}

		var filename string
		if sdk.Assets.Contains(name) {
			filename = name
		} else {
			filename = sdk.BinaryFilename(name, os, arch, variant)
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, filename))

		path := path.Join(api.Config.Download.Directory, filename)
		log.Debug(ctx, "downloading from %s", path)

		http.ServeFile(w, r, path)
		return nil
	}
}
