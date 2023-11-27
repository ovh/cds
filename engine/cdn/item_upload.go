package cdn

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/cdn"
	"github.com/ovh/cds/sdk/jws"
)

func (s *Service) verifySignatureFromRequest(ctx context.Context, r *http.Request) (*cdn.Signature, error) {
	signatureString := r.Header.Get("X-CDS-WORKER-SIGNATURE")
	var signature cdn.Signature
	if err := jws.UnsafeParse(signatureString, &signature); err != nil {
		return nil, err
	}

	if signature.Worker == nil {
		return nil, sdk.WrapError(sdk.ErrWrongRequest, "request is invalid. Missing worker data")
	}

	var privateKey []byte

	switch {
	case signature.RunJobID != "":
		var err error
		workerDataV2, err := s.getWorkerV2(ctx, signature.Worker.WorkerName, GetWorkerOptions{NeedPrivateKey: true})
		if err != nil {
			return nil, err
		}
		privateKey = workerDataV2.PrivateKey
	default:
		var err error
		workerData, err := s.getWorker(ctx, signature.Worker.WorkerName, GetWorkerOptions{NeedPrivateKey: true})
		if err != nil {
			return nil, err
		}
		privateKey = workerData.PrivateKey
	}

	// Verify Signature
	if err := jws.Verify(privateKey, signatureString, &signature); err != nil {
		return nil, sdk.WrapError(err, "worker key: %d", len(privateKey))
	}

	return &signature, nil
}

func (s *Service) postUploadHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		signature, err := s.verifySignatureFromRequest(ctx, r)
		if err != nil {
			return err
		}

		item, err := s.storeFile(ctx, *signature, r.Body, StoreFileOptions{})
		if err != nil {
			return err
		}
		return service.WriteJSON(w, item, http.StatusAccepted)
	}
}
