package repositories

import (
	"context"

	"github.com/ovh/cds/sdk"
)

// useBareAnalysisCache returns true when the operation only analyzes git data
// (signature, semver, changeset) and can run on the bare clones cache.
func (s *Service) useBareAnalysisCache(op sdk.Operation) bool {
	if !s.Cfg.BareAnalysisCache {
		return false
	}
	if op.Setup.Checkout.Branch == "" && op.Setup.Checkout.Tag == "" {
		return false
	}
	if op.LoadFiles.Pattern != "" || op.Setup.Push.FromBranch != "" {
		return false
	}
	return op.Setup.Checkout.CheckSignature || op.Setup.Checkout.GetChangeSet || op.Setup.Checkout.ProcessSemver
}

func (s *Service) processAnalyzeBare(ctx context.Context, op *sdk.Operation) error {
	return sdk.NewErrorFrom(sdk.ErrNotImplemented, "bare analysis cache processor")
}
