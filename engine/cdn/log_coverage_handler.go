package cdn

import (
	"context"
	"net/http"

	"github.com/ovh/cds/engine/cdn/item"
	"github.com/ovh/cds/engine/service"
	"github.com/ovh/cds/sdk"
)

// maxLogCoverageJobIDs bounds one call. The caller batches, and a batch that is too large turns a
// question about a window of jobs into a scan the CDN has to pay for while it is serving logs.
const maxLogCoverageJobIDs = 500

// postItemsLogCoverageHandler answers, for a batch of job identifiers, how many log items the CDN
// holds for each. It exists so that the API can tell apart a job whose logs were never stored from
// one whose logs it simply has not been asked for, which no counter on either side can say alone:
// the CDN knows the items it created, the API alone knows the jobs that ran.
//
// Read only, and never a log line: identifiers and cardinalities only.
func (s *Service) postItemsLogCoverageHandler() service.Handler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var req sdk.CDNJobLogCoverageRequest
		if err := service.UnmarshalBody(r, &req); err != nil {
			return err
		}

		if len(req.JobIDs) > maxLogCoverageJobIDs {
			return sdk.NewErrorFrom(sdk.ErrWrongRequest, "too many job identifiers given: %d, maximum is %d", len(req.JobIDs), maxLogCoverageJobIDs)
		}

		counts, stepCounts, err := item.CountLogItemsByJobIdentifiers(s.mustDBWithCtx(ctx), req.JobIDs)
		if err != nil {
			return err
		}

		return service.WriteJSON(w, sdk.CDNJobLogCoverageResponse{
			LogItemCountByJobID:     counts,
			StepLogItemCountByJobID: stepCounts,
		}, http.StatusOK)
	}
}
