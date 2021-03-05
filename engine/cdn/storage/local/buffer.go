package local

import (
	"context"
	"fmt"
	"os"

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

func (b *Buffer) Size(_ sdk.CDNItemUnit) (int64, error) {
	return b.size, nil
}

func (b *Buffer) BufferType() storage.CDNBufferType {
	return b.bufferType
}
