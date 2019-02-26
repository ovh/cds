package plugin

import (
	"io"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/action"
	"github.com/ovh/cds/engine/api/objectstore"
	"github.com/ovh/cds/sdk"
	"github.com/ovh/cds/sdk/log"
)

// AddBinary add binary to the plugin, uploading it to objectsore and updates databases
func AddBinary(db gorp.SqlExecutor, storage objectstore.Driver, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary, r io.ReadCloser) error {
	objectPath, err := storage.Store(b, r)
	if err != nil {
		return err
	}

	b.ObjectPath = objectPath
	p.Binaries = append(p.Binaries, *b)

	if p.Type == sdk.GRPCPluginAction {
		act, errA := action.LoadPublicAction(db, p.Name)
		if errA != nil {
			return sdk.WrapError(errA, "AddBinary> Cannot load public action for plugin type action")
		}

		// Add action requirement
		for _, req := range b.Requirements {
			found := false
			for _, reqAct := range act.Requirements {
				if req.Name == reqAct.Name && req.Type == reqAct.Type {
					found = true
					break
				}
			}
			if !found {
				if err := action.InsertActionRequirement(db, act.ID, req); err != nil {
					return sdk.WrapError(err, "Cannot insert action requirement %s", req.Name)
				}
			}
		}
	}

	return Update(db, p)
}

// UpdateBinary updates binary for the plugin, uploading it to objectsore and updates databases
func UpdateBinary(db gorp.SqlExecutor, storageDriver objectstore.Driver, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary, r io.ReadCloser) error {
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

	if err := storageDriver.Delete(oldBinary); err != nil {
		log.Error("UpdateBinary> unable to delete %+v", oldBinary)
	}

	objectPath, err := storageDriver.Store(b, r)
	if err != nil {
		return err
	}

	b.ObjectPath = objectPath
	p.Binaries[index] = *b

	if p.Type == sdk.GRPCPluginAction {
		act, errA := action.LoadPublicAction(db, p.Name)
		if errA != nil {
			return sdk.WrapError(errA, "AddBinary> Cannot load public action for plugin type action")
		}

		if err := action.DeleteActionRequirements(db, act.ID); err != nil {
			return sdk.WrapError(err, "Cannot delete requirements for action of plugin type action")
		}

		plgReq := sdk.Requirement{
			Name:  p.Name,
			Type:  sdk.PluginRequirement,
			Value: p.Name,
		}
		if err := action.InsertActionRequirement(db, act.ID, plgReq); err != nil {
			return sdk.WrapError(err, "Cannot insert plugin action requirement %s", plgReq.Name)
		}
		// Add action requirement
		for _, req := range b.Requirements {
			if err := action.InsertActionRequirement(db, act.ID, req); err != nil {
				return sdk.WrapError(err, "Cannot insert action requirement %s", req.Name)
			}
		}
	}

	return Update(db, p)
}
