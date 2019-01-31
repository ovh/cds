package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

func (api *API) getArtifactsStoreHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		return service.WriteJSON(w, objectstore.Instance(), http.StatusOK)
	}
}

func generateHash() (string, error) {
	size := 128
	bs := make([]byte, size)
	if _, err := rand.Read(bs); err != nil {
		return "", sdk.WrapError(err, "rand.Read failed")
	}
	str := hex.EncodeToString(bs)
	token := []byte(str)[0:size]

	log.Debug("api> generateHash> new generated id: %s", token)
	return string(token), nil
}
