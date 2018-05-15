package plugin

import (
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// AddBinary add binary to the plugin, uploading it to objectsore and updates databases
func AddBinary(db gorp.SqlExecutor, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary, r io.ReadCloser) error {
	objectPath, err := objectstore.Store(b, r)
	if err != nil {
		return err
	}

	b.ObjectPath = objectPath
	p.Binaries = append(p.Binaries, *b)

	return Update(db, p)
}

// UpdateBinary updates binary for the plugin, uploading it to objectsore and updates databases
func UpdateBinary(db gorp.SqlExecutor, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary, r io.ReadCloser) error {
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
		return sdk.ErrUnsupportedOSArchPlugin
	}

	if err := objectstore.Delete(oldBinary); err != nil {
		log.Error("UpdateBinary> unable to delete %+v", oldBinary)
	}

	objectPath, err := objectstore.Store(b, r)
	if err != nil {
		return err
	}

	b.ObjectPath = objectPath
	p.Binaries[index] = *b

	return Update(db, p)
}

// GetBinary returns the binary for selected os/arch
func GetBinary(p *sdk.GRPCPlugin, os, arch string) (*sdk.GRPCPluginBinary, io.ReadCloser, error) {
	var b *sdk.GRPCPluginBinary
	for i := range p.Binaries {
		if p.Binaries[i].OS == os && p.Binaries[i].Arch == arch {
			b = &p.Binaries[i]
			break
		}
	}

	if b == nil {
		return nil, nil, sdk.ErrUnsupportedOSArchPlugin
	}

	buf, err := objectstore.Fetch(*b)
	if err != nil {
		return nil, nil, sdk.WrapError(err, "GetBinary")
	}

	return b, buf, nil
}

// FetchBinary returns a download URL for a binary
func FetchBinary(p *sdk.GRPCPlugin, os, arch string) (*sdk.GRPCPluginBinary, string, error) {
	var b *sdk.GRPCPluginBinary
	for i := range p.Binaries {
		if p.Binaries[i].OS == os && p.Binaries[i].Arch == arch {
			b = &p.Binaries[i]
			break
		}
	}

	if b == nil {
		return nil, "", sdk.ErrUnsupportedOSArchPlugin
	}

	url, err := objectstore.FetchTempURL(*b)
	if err != nil {
		return nil, "", sdk.WrapError(err, "FetchBinary")
	}

	return b, url, nil
}
