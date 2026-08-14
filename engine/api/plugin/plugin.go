package plugin

import (
	"context"
	"io"

	"github.com/go-gorp/gorp"
	"github.com/rockbears/log"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
)

// UploadBinary uploads the content of a plugin binary to the storage, without referencing it in
// database. The object is written under a key derived from the binary name, os and arch: a binary
// being replaced is only overwritten when the upload completes, so the plugin stays downloadable
// for the whole upload.
func UploadBinary(ctx context.Context, storageDriver objectstore.Driver, b *sdk.GRPCPluginBinary, r io.ReadCloser) error {
	objectPath, err := storageDriver.Store(b, r)
	if err != nil {
		return err
	}
	b.ObjectPath = objectPath
	return nil
}

// SetBinary references an uploaded binary in the plugin, replacing the binary for the same os and
// arch if any, and updates the plugin in database.
// When the replaced binary was not stored under the same key as the new one, its object is not
// referenced anymore and is returned: the caller must delete it with DeleteStaleBinary, once the
// transaction is committed.
// The plugin must have been loaded with LoadByNameForUpdate: all the binaries of a plugin are
// stored in a single row.
func SetBinary(db gorp.SqlExecutor, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary) (*sdk.GRPCPluginBinary, error) {
	var staleBinary *sdk.GRPCPluginBinary
	index := -1
	for i := range p.Binaries {
		if p.Binaries[i].OS == b.OS && p.Binaries[i].Arch == b.Arch {
			index = i
			if old := p.Binaries[i]; !sameStorageKey(old, *b) {
				staleBinary = &old
			}
			break
		}
	}

	if index >= 0 {
		p.Binaries[index] = *b
	} else {
		p.Binaries = append(p.Binaries, *b)
	}

	if err := Update(db, p); err != nil {
		return nil, err
	}

	return staleBinary, nil
}

// DeleteStaleBinary removes from the storage a binary that is not referenced anymore. It never
// fails the caller: the object is only leaked in the storage.
func DeleteStaleBinary(ctx context.Context, storageDriver objectstore.Driver, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary) {
	if err := storageDriver.Delete(ctx, b); err != nil {
		log.ErrorWithStackTrace(ctx, sdk.WrapError(err, "unable to delete stale binary %s of plugin %s (%s/%s)", b.Name, p.Name, b.OS, b.Arch))
	}
}

// sameStorageKey returns true when both binaries are stored as the same object, ie. when storing
// one overwrites the other.
func sameStorageKey(a, b sdk.GRPCPluginBinary) bool {
	return a.GetPath() == b.GetPath() && a.GetName() == b.GetName()
}

// DeleteBinary removes a binary for the plugin in database and returns it, so that the caller can
// delete its object with DeleteStaleBinary once the transaction is committed.
// The plugin must have been loaded with LoadByNameForUpdate.
func DeleteBinary(db gorp.SqlExecutor, p *sdk.GRPCPlugin, os, arch string) (*sdk.GRPCPluginBinary, error) {
	var oldBinary *sdk.GRPCPluginBinary
	filteredBinaries := make(sdk.GRPCPluginBinaries, 0, len(p.Binaries))
	for i := range p.Binaries {
		if p.Binaries[i].OS == os && p.Binaries[i].Arch == arch {
			old := p.Binaries[i]
			oldBinary = &old
		} else {
			filteredBinaries = append(filteredBinaries, p.Binaries[i])
		}
	}
	if oldBinary == nil {
		return nil, sdk.WithStack(sdk.ErrUnsupportedOSArchPlugin)
	}

	p.Binaries = filteredBinaries
	if err := Update(db, p); err != nil {
		return nil, err
	}

	return oldBinary, nil
}
