package services

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/sdk"
)

func GetCDNPublicTCPAddress(ctx context.Context, db gorp.SqlExecutor) (string, bool, error) {
	srvs, err := LoadAllByType(ctx, db, sdk.TypeCDN)
	if err != nil {
		return "", false, err
	}

	var tcp_addr string
	var tcp_tls bool

findAddr:
	for _, svr := range srvs {
		if addr, ok := svr.Config["public_tcp"]; ok {
			tcp_addr = addr.(string)
			if tls, ok := svr.Config["public_tcp_enable_tls"]; ok {
				tcp_tls = tls.(bool)
			}
			break findAddr
		}
	}
	if tcp_addr != "" {
		return tcp_addr, tcp_tls, nil
	}
	return "", false, sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find any tcp configuration in CDN Uservice")
}

func GetCDNPublicHTTPAddress(ctx context.Context, db gorp.SqlExecutor) (string, error) {
	srvs, err := LoadAllByType(ctx, db, sdk.TypeCDN)
	if err != nil {
		return "", err
	}
	for _, svr := range srvs {
		if addr, ok := svr.Config["public_http"]; ok {
			return addr.(string), nil
		}
	}
	return "", sdk.NewErrorFrom(sdk.ErrNotFound, "unable to find any http configuration in CDN Uservice")
}
