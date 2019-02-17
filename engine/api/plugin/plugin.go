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
func AddBinary(db gorp.SqlExecutor, p *sdk.GRPCPlugin, b *sdk.GRPCPluginBinary, r io.ReadCloser) error {
	objectPath, err := objectstore.Store(b, r)
	if err != nil {
		return err
	}

	b.ObjectPath = objectPath
	p.Binaries = append(p.Binaries, *b)

	if p.Type == sdk.GRPCPluginAction {
		act, errA := action.LoadTypePluginByName(db, p.Name)
		if errA != nil {
			return sdk.WrapError(errA, "cannot load public action for plugin type action")
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
				req.ActionID = act.ID
				if err := action.InsertRequirement(db, &req); err != nil {
					return sdk.WrapError(err, "cannot insert action requirement %s", req.Name)
				}
			}
		}
	}

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

	if p.Type == sdk.GRPCPluginAction {
		act, errA := action.LoadTypePluginByName(db, p.Name)
		if errA != nil {
			return sdk.WrapError(errA, "cannot load public action for plugin type action")
		}

		if err := action.DeleteRequirementsByActionID(db, act.ID); err != nil {
			return sdk.WrapError(err, "cannot delete requirements for action of plugin type action")
		}

		if err := action.InsertRequirement(db, &sdk.Requirement{
			ActionID: act.ID,
			Name:     p.Name,
			Type:     sdk.PluginRequirement,
			Value:    p.Name,
		}); err != nil {
			return sdk.WrapError(err, "cannot insert plugin action requirement %s", p.Name)
		}

		// add action requirement
		for _, req := range b.Requirements {
			req.ActionID = act.ID
			if err := action.InsertRequirement(db, &req); err != nil {
				return sdk.WrapError(err, "cannot insert action requirement %s", req.Name)
			}
		}
	}

	return Update(db, p)
}
