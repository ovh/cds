package cdn

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/jws"
)

func (s *Service) postUploadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		signatureString := r.Header.Get("X-CDS-WORKER-SIGNATURE")
		var signature cdn.Signature
		if err := jws.UnsafeParse(signatureString, &signature); err != nil {
			return err
		}

		if signature.Worker == nil {
			return sdk.WrapError(sdk.ErrWrongRequest, "request is invalid. Missing worker data")
		}
		workerData, err := s.getWorker(ctx, signature.Worker.WorkerName, GetWorkerOptions{NeedPrivateKey: true})
		if err != nil {
			return err
		}

		// Verify Signature
		if err := jws.Verify(workerData.PrivateKey, signatureString, &signature); err != nil {
			return sdk.WrapError(err, "worker key: %d", len(workerData.PrivateKey))
		}

		if err := s.storeFile(ctx, signature, r.Body, StoreFileOptions{}); err != nil {
			return err
		}
		return nil
	}
}
