package services

import (
	"context"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/lib/pq"
	"github.com/ovh/cds/engine/api/database/gorpmapping"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc is used as options to load services
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.Service) error

// LoadOptions provides all options on service loads functions
var LoadOptions = struct {
	WithStatus LoadOptionFunc
}{
	WithStatus: loadServiceStatus,
}

func loadServiceStatus(ctx context.Context, db gorp.SqlExecutor, services ...*sdk.Service) error {
	var servicesIDs []int64
	for _, s := range services {
		servicesIDs = append(servicesIDs, s.ID)
	}

	ss, err := loadAllServiceStatus(ctx, db, servicesIDs)
	if err != nil {
		return err
	}
	for i := range services {
		srv := services[i]
		srv.MonitoringStatus = sdk.MonitoringStatus{Now: time.Now()}
		completeStatus(ss, srv)
	}

	return nil
}

// loadAllServiceStatus returns all services status
func loadAllServiceStatus(ctx context.Context, db gorp.SqlExecutor, servicesIDs []int64) ([]sdk.ServiceStatus, error) {
	query := gorpmapping.NewQuery(`SELECT * FROM service_status WHERE service_id = ANY($1)`).Args(pq.Int64Array(servicesIDs))
	ss := []sdk.ServiceStatus{}
	if err := gorpmapping.GetAll(ctx, db, query, &ss); err != nil {
		return nil, sdk.WrapError(err, "cannot get services")
	}
	return ss, nil
}

func completeStatus(ss []sdk.ServiceStatus, srv *sdk.Service) {
	for _, status := range ss {
		if srv.ID == status.ServiceID {
			for _, line := range status.MonitoringStatus.Lines {
				if status.SessionID != nil {
					line.SessionID = *status.SessionID
				}
				if srv.ConsumerID != nil {
					line.ConsumerID = *srv.ConsumerID
				}
				srv.MonitoringStatus.Lines = append(srv.MonitoringStatus.Lines, line)
			}
		}
	}
}
