package cdn

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/services"
	"github.com/ovh/cds/sdk"
)

const (
	ParamRunID      = "runid"
	ParamProjectKey = "projectkey"
	ParamCacheTag   = "cachetag"
)

func ListItems(ctx context.Context, db gorp.SqlExecutor, itemtype sdk.CDNItemType, params map[string]string) (sdk.CDNItemLinks, error) {
	var result sdk.CDNItemLinks

	if len(params) == 0 {
		return result, sdk.WrapError(sdk.ErrInvalidData, "need parameters to filter items")
	}

	srvs, err := services.LoadAllByType(ctx, db, sdk.TypeCDN)
	if err != nil {
		return result, err
	}
	if len(srvs) == 0 {
		return result, sdk.WrapError(sdk.ErrNotFound, "no service found")
	}

	path := fmt.Sprintf("/item/%s?", itemtype)
	firstParams := true
	for k, v := range params {
		if firstParams {
			path = fmt.Sprintf("%s%s=%s", path, k, url.QueryEscape(v))
			firstParams = false
		} else {
			path = fmt.Sprintf("%s&%s=%s", path, k, url.QueryEscape(v))
		}

	}
	btes, _, code, err := services.DoRequest(ctx, srvs, http.MethodGet, path, nil)
	if code == http.StatusNotFound {
		return result, sdk.WithStack(sdk.ErrNotFound)
	}
	if err != nil {
		return result, err
	}
	var cdnItems []sdk.CDNItem
	if err := sdk.JSONUnmarshal(btes, &cdnItems); err != nil {
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
