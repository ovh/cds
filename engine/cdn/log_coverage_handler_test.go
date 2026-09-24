package cdn

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ovh/cds/sdk"
)

// TestPostItemsLogCoverageHandler_BatchCap pins that an oversized batch is refused before anything
// touches the database. The cap is the contract the caller batches against, so a silent acceptance
// would turn a bounded question into a scan the CDN pays for while it is serving logs.
func TestPostItemsLogCoverageHandler_BatchCap(t *testing.T) {
	s := &Service{}

	jobIDs := make([]string, maxLogCoverageJobIDs+1)
	for i := range jobIDs {
		jobIDs[i] = sdk.UUID()
	}
	body, err := json.Marshal(sdk.CDNJobLogCoverageRequest{JobIDs: jobIDs})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/admin/items/log-coverage", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	err = s.postItemsLogCoverageHandler()(context.TODO(), httptest.NewRecorder(), req)
	require.Error(t, err)
	require.Equal(t, sdk.ErrWrongRequest.ID, sdk.ExtractHTTPError(err).ID)
}
