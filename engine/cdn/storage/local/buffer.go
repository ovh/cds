package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/cdn/storage"
	"github.com/ovh/cds/engine/cdn/storage/encryption"
	"github.com/ovh/cds/sdk"
)

var (
	_ storage.FileBufferUnit = new(Buffer)
)

type Buffer struct {
	AbstractLocal
	config storage.LocalBufferConfiguration
	encryption.NoConvergentEncryption
	bufferType storage.CDNBufferType
}

const driverBufferName = "local-buffer"

func init() {
	storage.RegisterDriver(driverBufferName, new(Buffer))
}

func (b *Buffer) GetDriverName() string {
	return driverBufferName
}

func (b *Buffer) Init(ctx context.Context, cfg interface{}, bufferType storage.CDNBufferType) error {
	config, is := cfg.(*storage.LocalBufferConfiguration)
	if !is {
		return sdk.WithStack(fmt.Errorf("invalid configuration: %T", cfg))
	}
	b.path = config.Path
	b.config = *config
	b.NoConvergentEncryption = encryption.NewNoConvergentEncryption(config.Encryption)
	b.bufferType = bufferType
	if err := os.MkdirAll(b.config.Path, os.FileMode(0700)); err != nil {
		return sdk.WithStack(err)
	}

	b.GoRoutines.Run(ctx, "cdn-local-compute-size", func(ctx context.Context) {
		b.computeSize(ctx)
	})
	b.isBuffer = true
	return nil
}

func (b *Buffer) Size(_ context.Context, _ sdk.CDNItemUnit) (int64, error) {
	return b.size, nil
}

func (b *Buffer) BufferType() storage.CDNBufferType {
	return b.bufferType
}

func (b *Buffer) ResyncWithDatabase(ctx context.Context, db gorp.SqlExecutor, t sdk.CDNItemType, dryRun bool) {
	root := fmt.Sprintf("%s/%s", b.config.Path, string(t))
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info == nil {
			return nil
		}
		if info.IsDir() {
			log.Warn(ctx, "local-buffer: found directory inside %s: %s", string(t), path)
			return nil
		}
		_, fileName := filepath.Split(path)
		has, err := storage.HashItemUnitByApiRefHash(db, fileName, b.ID())
		if err != nil {
			log.Error(ctx, "local-buffer: unable to check if unit item exist for api ref hash %s: %v", fileName, err)
			return nil
		}
		if has {
			return nil
		}
		if !dryRun {
			if err := os.Remove(path); err != nil {
				log.Error(ctx, "local-buffer: unable to remove file %s: %v", path, err)
				return nil
			}
			log.Info(ctx, "local-buffer: file %s has been deleted", fileName)
		} else {
			log.Info(ctx, "local-buffer: file %s should be deleted", fileName)
		}
		return nil
	}); err != nil {
		log.Error(ctx, "local-buffer: error during walk operation: %v", err)
	}

}
