package plugin

import (
	"context"
	"io"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
)

// AddBinary add binary to the plugin, uploading it to objectsore and updates databases
func AddBinary(ctx context.Context, db gorp.SqlExecutor, storage objectstore.Driver, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary, r io.ReadCloser) error {
	objectPath, err := storage.Store(b, r)
	if err != nil {
		return err
	}

	b.ObjectPath = objectPath
	p.Binaries = append(p.Binaries, *b)

	return Update(db, p)
}

// UpdateBinary updates binary for the plugin, uploading it to objectsore and updates databases
func UpdateBinary(ctx context.Context, db gorp.SqlExecutor, storageDriver objectstore.Driver, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary, r io.ReadCloser) error {
	var oldBinary *sdk.GRPCPluginBinary
	var index int
	for i := range p.Binaries {
		if p.Binaries[i].OS == b.OS && p.Binaries[i].Arch == b.Arch {
			oldBinary = &p.Binaries[i]
			index = i
			break
		}
	}
	if oldBinary == nil {
		return sdk.WithStack(sdk.ErrUnsupportedOSArchPlugin)
	}

	if err := storageDriver.Delete(ctx, oldBinary); err != nil {
		log.Error(ctx, "UpdateBinary> unable to delete %+v", oldBinary)
	}

	objectPath, err := storageDriver.Store(b, r)
	if err != nil {
		return err
	}

	b.ObjectPath = objectPath
	p.Binaries[index] = *b

	return Update(db, p)
}

// DeleteBinary remove a binary for the plugin from objectsore and updates databases.
func DeleteBinary(ctx context.Context, db gorp.SqlExecutor, storageDriver objectstore.Driver, p *sdk.GRPCPlugin, os, arch string) error {
	var oldBinary *sdk.GRPCPluginBinary
	filteredBinaries := make(sdk.GRPCPluginBinaries, 0, len(p.Binaries))
	for i := range p.Binaries {
		if p.Binaries[i].OS == os && p.Binaries[i].Arch == arch {
			oldBinary = &p.Binaries[i]
		} else {
			filteredBinaries = append(filteredBinaries, p.Binaries[i])
		}
	}
	if oldBinary == nil {
		return sdk.WithStack(sdk.ErrUnsupportedOSArchPlugin)
	}

	if err := storageDriver.Delete(ctx, oldBinary); err != nil {
		log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to delete plugin %s binary %s/%s", p.ID, os, arch))
	}

	p.Binaries = filteredBinaries
	return Update(db, p)
}
