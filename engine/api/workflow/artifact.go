package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
)

func ListArtifacts(ctx context.Context, db gorp.SqlExecutor, runID int64) (sdk.CDNItemLinks, error) {
	var result sdk.CDNItemLinks

	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeCDN)
	if err != nil {
		return result, err
	}
	if len(srvs) == 0 {
		return result, sdk.WrapError(sdk.ErrNotFound, "no service found")
	}

	path := fmt.Sprintf("/service/item/%s?runid=%d", sdk.CDNTypeItemArtifact, runID)
	btes, _, _, err := services.DoRequest(ctx, db, srvs, http.MethodGet, path, nil)
	if err != nil {
		return result, err
	}
	var cdnItems []sdk.CDNItem
	if err := json.Unmarshal(btes, &cdnItems); err != nil {
		return result, sdk.WithStack(err)
	}

	httpURL, err := services.GetCDNPublicHTTPAdress(ctx, db)
	if err != nil {
		return result, err
	}

	result.CDNHttpURL = httpURL
	result.Items = cdnItems
	return result, nil
}
