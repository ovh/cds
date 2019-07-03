package sdk

// SharedInfraGroupName is the name of the builtin group used to share infrastructure between projects
const SharedInfraGroupName = "shared.infra"

// Group represent a group of user.
type Group struct {
	ID      int64  `json:"id" yaml:"-" db:"id"`
	Name    string `json:"name" yaml:"name" cli:"name,key" db:"name"`
	Members []User `json:"members,omitempty" yaml:"members,omitempty" db:"-"`
}

type Groups []Group

func (groups Groups) HasOneOf(groupIDs ...int64) bool {
	ids := GroupsToIDs(groups)
	for _, id := range groupIDs {
		if IsInInt64Array(id, ids) {
			return true
		}
	}
	return false
}

// GroupPermission represent a group and his role in the project
type GroupPermission struct {
	Group      Group `json:"group"`
	Permission int   `json:"permission"`
}

// ProjectGroup represent a link with a project
type ProjectGroup struct {
	Project    Project `json:"project"`
	Permission int     `json:"permission"`
}

// WorkflowGroup represents the permission to a workflow
type WorkflowGroup struct {
	Workflow   Workflow `json:"workflow"`
	Permission int      `json:"permission"`
}

// GroupsToIDs returns ids of given groups.
func GroupsToIDs(gs []Group) []int64 {
	ids := make([]int64, len(gs))
	for i := range gs {
		ids[i] = gs[i].ID
	}
	return ids
}

// GroupPointersToIDs returns ids of given groups.
func GroupPointersToIDs(gs []*Group) []int64 {
	ids := make([]int64, len(gs))
	for i := range gs {
		ids[i] = gs[i].ID
	}
	return ids
}

// IsMember checks if given group memeber is part of current group.
func (g Group) IsMember(groupIDs []int64) bool {
	for _, id := range groupIDs {
		if id == g.ID {
			return true
		}
	}
	return false
}

// IsAdmin checks if given authentified user is admin for current group,
// group should have members aggregated and authentified user old user struct should be set.
func (g Group) IsAdmin(u AuthentifiedUser) bool {
	for _, member := range g.Members {
		if member.ID == u.OldUserStruct.ID {
			return member.GroupAdmin
		}
	}
	return false
}
