package swift

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/ncw/swift"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/engine/gorpmapper"
	"github.com/ovh/cds/sdk"
)

type Swift struct {
	client swift.Connection
	storage.AbstractUnit
	encryption.ConvergentEncryption
	config storage.SwiftStorageConfiguration
}

var (
	_ storage.StorageUnit = new(Swift)
)

const driverName = "swift"

func init() {
	storage.RegisterDriver(driverName, new(Swift))
}

func (s *Swift) GetDriverName() string {
	return driverName
}

func (s *Swift) Init(_ context.Context, cfg interface{}) error {
	config, is := cfg.(*storage.SwiftStorageConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	s.config = *config
	s.ConvergentEncryption = encryption.New(config.Encryption)
	s.client = swift.Connection{
		AuthUrl:        config.Address,
		Region:         config.Region,
		Tenant:         config.Tenant,
		Domain:         config.Domain,
		UserName:       config.Username,
		ApiKey:         config.Password,
		ConnectTimeout: time.Minute * 1,
		Timeout:        time.Minute * 10,
	}
	return sdk.WithStack(s.client.Authenticate())
}

func (s *Swift) ItemExists(ctx context.Context, m *gorpmapper.Mapper, db gorp.SqlExecutor, i sdk.CDNItem) (bool, error) {
	iu, err := s.ExistsInDatabase(ctx, m, db, i.ID)
	if err != nil {
		if sdk.ErrorIs(err, sdk.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	container, object := s.getItemPath(*iu)
	allObjs, _ := s.client.ObjectNamesAll(container, nil)
	for i := range allObjs {
		if allObjs[i] == object {
			return true, nil
		}
	}
	return false, nil
}

func (s *Swift) NewWriter(ctx context.Context, i sdk.CDNItemUnit) (io.WriteCloser, error) {
	container, object := s.getItemPath(i)

	if err := s.client.ContainerCreate(container, nil); err != nil {
		return nil, sdk.WrapError(err, "Unable to create container %s", container)
	}
	log.Debug(ctx, "[%T] writing to %s/%s", s, container, object)
	file, err := s.client.ObjectCreate(container, object, true, "", "application/octet-stream", nil)
	if err != nil {
		return nil, sdk.WrapError(err, "SwiftStore> Unable to create object %s", object)
	}

	return file, nil
}

func (s *Swift) NewReader(ctx context.Context, i sdk.CDNItemUnit) (io.ReadCloser, error) {
	container, object := s.getItemPath(i)
	log.Debug(ctx, "[%T] reading from %s/%s", s, container, object)
	file, _, err := s.client.ObjectOpen(container, object, true, nil)
	if err != nil {
		return nil, sdk.WithStack(err)
	}
	return file, nil
}

func (s *Swift) getItemPath(i sdk.CDNItemUnit) (container string, object string) {
	loc := i.Locator
	container = fmt.Sprintf("%s-%s-%s", s.config.ContainerPrefix, i.Item.Type, loc[:3])
	object = loc
	container, object = escape(container, object)
	return container, object
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
	return []sdk.MonitoringStatusLine{{
		Component: "backend/" + s.Name(),
		Value:     fmt.Sprintf("Swift OK (%d containers, %d objects, %d bytes used", info.Containers, info.Objects, info.BytesUsed),
		Status:    sdk.MonitoringStatusOK,
	}}
}

func (s *Swift) Remove(ctx context.Context, i sdk.CDNItemUnit) error {
	container, object := s.getItemPath(i)
	if err := s.client.ObjectDelete(container, object); err != nil {
		if strings.Contains(err.Error(), "Object Not Found") {
			return sdk.ErrNotFound
		}
		return sdk.WithStack(err)
	}
	return nil
}

func (s *Swift) ResyncWithDatabase(ctx context.Context, _ gorp.SqlExecutor, _ sdk.CDNItemType, _ bool) {
	log.Error(ctx, "Resynchronization with database not implemented for swift storage unit")
}
