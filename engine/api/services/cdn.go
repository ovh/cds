package services

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func GetCDNPublicTCPAdress(ctx context.Context, db gorp.SqlExecutor) (string, error) {
	srvs, err := LoadAllByType(ctx, db, sdk.TypeCDN)
	if err != nil {
		return "", err
	}
	for _, svr := range srvs {
		if addr, ok := svr.Config["public_tcp"]; ok {
			return addr.(string), nil
		}
	}
	return "", sdk.WrapError(sdk.ErrNotFound, "unable to find any tcp configuration in CDN Uservice")
}
