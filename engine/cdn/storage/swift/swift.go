package swift

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/go-gorp/gorp"
	"github.com/ncw/swift"
	"go.opencensus.io/stats"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
	"github.com/ovh/cds/sdk/telemetry"
)

type Swift struct {
	client swift.Connection
	storage.AbstractUnit
	encryption.ConvergentEncryption
	config storage.SwiftStorageConfiguration
}

var (
	_                 storage.StorageUnit = new(Swift)
	metricsContainers                     = stats.Int64("cdn/storage/swift/containers", "nb containers", stats.UnitDimensionless)
	metricsObjects                        = stats.Int64("cdn/storage/swift/objects", "nb objects", stats.UnitDimensionless)
	metricsSize                           = stats.Int64("cdn/storage/swift/size", "swift bytes used", stats.UnitDimensionless)
	metricsReaders                        = stats.Int64("cdn/storage/swift/readers", "nb readers", stats.UnitDimensionless)
	metricsWriters                        = stats.Int64("cdn/storage/swift/writers", "nb writers", stats.UnitDimensionless)
)

func init() {
	storage.RegisterDriver("swift", new(Swift))
}

func (s *Swift) Init(ctx context.Context, _ *sdk.GoRoutines, cfg interface{}) error {
	config, is := cfg.(*storage.SwiftStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	s.ConvergentEncryption = encryption.New(config.Encryption)
	s.client = swift.Connection{
		AuthUrl:  config.Address,
		Region:   config.Region,
		Tenant:   config.Tenant,
		Domain:   config.Domain,
		UserName: config.Username,
		ApiKey:   config.Password,
	}

	if err := telemetry.InitMetricsInt64(ctx, metricsContainers, metricsObjects, metricsSize, metricsReaders, metricsWriters); err != nil {
		return err
	}
	return nil
}

func (s *Swift) ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	iu, err := s.ExistsInDatabase(ctx, m, db, i.ID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	container, object, err := s.getItemPath(*iu)
	if err != nil {
		return false, err
	}

	allObjs, _ := s.client.ObjectNamesAll(container, nil)
	for i := range allObjs {
		if allObjs[i] == object {
			return true, nil
		}
	}
	return false, nil
}

func (s *Swift) NewWriter(ctx context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error) {
	container, object, err := s.getItemPath(i)
	if err != nil {
		return nil, err
	}

	if err := s.client.ContainerCreate(container, nil); err != nil {
		return nil, sdk.WrapError(err, "Unable to create container %s", container)
	}

	file, err := s.client.ObjectCreate(container, object, true, "", "application/octet-stream", nil)
	if err != nil {
		return nil, sdk.WrapError(err, "SwiftStore> Unable to create object %s", object)
	}

	telemetry.Record(ctx, metricsWriters, 1)
	return file, nil
}

func (s *Swift) NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	container, object, err := s.getItemPath(i)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	go func() {
		if _, err = s.client.ObjectGet(container, object, pw, true, nil); err != nil {
			log.Error(context.Background(), "unable to get object %s/%s: %v", container, object, err)
			return
		}
		if err := pw.Close(); err != nil {
			log.Error(context.Background(), "unable to close pipewriter %s/%s: %v", container, object, err)
			return
		}
	}()

	telemetry.Record(ctx, metricsReaders, 1)
	return pr, nil
}

func (s *Swift) getItemPath(i sdk.CDNItemUnit) (container string, object string, err error) {
	loc := i.Locator
	container = fmt.Sprintf("%s-%s-%s", s.config.ContainerPrefix, i.Item.Type, loc[:3])
	object = loc
	container, object = escape(container, object)
	return container, object, nil
}

func escape(container, object string) (string, string) {
	container = url.QueryEscape(container)
	container = strings.Replace(container, "/", "-", -1)
	object = url.QueryEscape(object)
	object = strings.Replace(object, "/", "-", -1)
	return container, object
}

// Status returns the status of swift account
func (s *Swift) Status(ctx context.Context) []sdk.MonitoringStatusLine {
	info, _, err := s.client.Account()
	if err != nil {
		return []sdk.MonitoringStatusLine{{Component: "backend/" + s.Name(), Value: "Swift KO" + err.Error(), Status: sdk.MonitoringStatusAlert}}
	}
	telemetry.Record(ctx, metricsContainers, info.Containers)
	telemetry.Record(ctx, metricsObjects, info.Objects)
	telemetry.Record(ctx, metricsSize, info.BytesUsed)
	return []sdk.MonitoringStatusLine{{
		Component: "backend/" + s.Name(),
		Value:     fmt.Sprintf("Swift OK (%d containers, %d objects, %d bytes used", info.Containers, info.Objects, info.BytesUsed),
		Status:    sdk.MonitoringStatusOK,
	}}
}
