package api

import (
	"context"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
)

func (api *API) pushCacheHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["key"]
		tag := vars["tag"]

		if r.Body == nil {
			return sdk.ErrWrongRequest
		}

		// btes, err := ioutil.ReadAll(r.Body)
		// if err != nil {
		// 	log.Error("postWorkflowPushHandler> Unable to read body: %v", err)
		// 	return sdk.ErrWrongRequest
		// }
		// defer r.Body.Close()
		//
		// tr := tar.NewReader(bytes.NewReader(btes))

		cacheObject := sdk.Cache{
			Name:    "cache.tar.gz",
			Project: projectKey,
			Tag:     tag,
		}
		// for {
		// 	hdr, err := tr.Next()
		// 	if err == io.EOF {
		// 		break
		// 	}
		// 	// buf.WriteString()
		// 	fmt.Println("hdr", hdr)
		// 	// buff := new(bytes.Buffer)
		// 	trc := ioutil.NopCloser(tr)
		//
		// 	if _, err := io.Copy(os.Stdout, trc); err != nil {
		// 		err = sdk.NewError(sdk.ErrWrongRequest, fmt.Errorf("Unable to read tar file"))
		// 		fmt.Println(err)
		// 	}
		//
		// 	// b := buff.Bytes()
		// 	// fmt.Println(string(b))
		//
		// }
		_, errO := objectstore.StoreArtifact(&cacheObject, r.Body)
		if errO != nil {
			r.Body.Close()
			return sdk.WrapError(errO, "SaveFile>Cannot store artifact")
		}
		r.Body.Close()

		return nil
	}
}

func (api *API) pullCacheHandler() Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		vars := mux.Vars(r)
		projectKey := vars["permProjectKey"]
		tag := vars["tag"]

		cacheObject := sdk.Cache{
			Project: projectKey,
			Tag:     tag,
		}

		f, err := objectstore.FetchArtifact(&cacheObject)
		if err != nil {
			return sdk.WrapError(err, "pullCacheHandler> Cannot fetch cache object")
		}

		if _, err := io.Copy(w, f); err != nil {
			_ = f.Close()
			return sdk.WrapError(err, "pullCacheHandler> Cannot stream cache file")
		}

		if err := f.Close(); err != nil {
			return sdk.WrapError(err, "pullCacheHandler> Cannot close cache file")
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Content-Disposition", "attachment; filename=\"cache.tar.gz\"")
		return nil
	}
}
