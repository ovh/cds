package group

import (
	"context"

	"github.com/go-gorp/gorp"

	"github.com/ovh/cds/engine/api/user"
	"github.com/ovh/cds/sdk"
)

// LoadOptionFunc for group.
type LoadOptionFunc func(context.Context, gorp.SqlExecutor, ...*sdk.Group) error

// LoadOptions provides all options on group loads functions.
var LoadOptions = struct {
	Default          LoadOptionFunc
	WithMembers      LoadOptionFunc
	WithOrganization LoadOptionFunc
}{
	Default:          loadDefault,
	WithMembers:      loadMembers,
	WithOrganization: loadOrganization,
}

func loadDefault(ctx context.Context, db gorp.SqlExecutor, gs ...*sdk.Group) error {
	if err := loadMembers(ctx, db, gs...); err != nil {
		return err
	}
	return loadOrganization(ctx, db, gs...)
}

func loadMembers(ctx context.Context, db gorp.SqlExecutor, gs ...*sdk.Group) error {
	groupIDs := sdk.GroupPointersToIDs(gs)

	// Get all links group user for group ids
	links, err := LoadLinksGroupUserForGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}
	mLinks := make(map[int64][]LinkGroupUser)
	for i := range links {
		if _, ok := mLinks[links[i].GroupID]; !ok {
			mLinks[links[i].GroupID] = []LinkGroupUser{links[i]}
		} else {
			mLinks[links[i].GroupID] = append(mLinks[links[i].GroupID], links[i])
		}
	}

	// Get all authentified users for migrations
	members, err := user.LoadAllByIDs(ctx, db, links.ToUserIDs(), user.LoadOptions.WithOrganization)
	if err != nil {
		return err
	}
	mMembers := members.ToMapByID()

	// Set members on each groups
	for _, g := range gs {
		if _, ok := mLinks[g.ID]; ok {
			g.Members = make([]sdk.GroupMember, 0, len(mLinks[g.ID]))
			for _, link := range mLinks[g.ID] {
				if member, ok := mMembers[link.AuthentifiedUserID]; ok {
					g.Members = append(g.Members, sdk.GroupMember{
						ID:           member.ID,
						Username:     member.Username,
						Fullname:     member.Fullname,
						Admin:        link.Admin,
						Organization: member.Organization,
					})
				}
			}
		}
	}

	return nil
}

func loadOrganization(ctx context.Context, db gorp.SqlExecutor, gs ...*sdk.Group) error {
	groupIDs := sdk.GroupPointersToIDs(gs)

	// Get all organizations for group ids
	orgs, err := LoadOrganizationsByGroupIDs(ctx, db, groupIDs)
	if err != nil {
		return err
	}
	mOrgs := make(map[int64]Organization)
	for i := range orgs {
		mOrgs[orgs[i].GroupID] = orgs[i]
	}

	// Set organization on each groups
	for i := range gs {
		if org, ok := mOrgs[gs[i].ID]; ok {
			gs[i].Organization = org.Organization
		}
	}

	return nil
}

// LoadLinkGroupProjectOptionFunc for link group project.
type LoadLinkGroupProjectOptionFunc func(context.Context, gorp.SqlExecutor, ...*LinkGroupProject) error

// LoadLinkGroupProjectOptions provides all options on link group project loads functions.
var LoadLinkGroupProjectOptions = struct {
	WithGroups LoadLinkGroupProjectOptionFunc
}{
	WithGroups: loadLinkGroupProjectGroups,
}

func loadLinkGroupProjectGroups(ctx context.Context, db gorp.SqlExecutor, gps ...*LinkGroupProject) error {
	groupIDs := make(sdk.Int64Slice, len(gps))
	for i := range gps {
		groupIDs[i] = gps[i].GroupID
	}
	groupIDs.Unique()

	gs, err := LoadAllByIDs(ctx, db, groupIDs, LoadOptions.WithOrganization)
	if err != nil {
		return err
	}

	m := gs.ToMap()
	for i := range gps {
		gps[i].Group = m[gps[i].GroupID]
	}

	return nil
}
