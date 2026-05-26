package sdk

import (
	"sort"
	"time"
)

const (
	// Global Role
	GlobalRoleManagePermission   = "manage-permission"
	GlobalRoleManageOrganization = "manage-organization"
	GlobalRoleManageRegion       = "manage-region"
	GlobalRoleManageHatchery     = "manage-hatchery"
	GlobalRoleManageUser         = "manage-user"
	GlobalRoleManageGroup        = "manage-group"
	GlobalRoleManagePlugin       = "manage-plugin"
	GlobalRoleProjectCreate      = "create-project"

	// Project Role
	ProjectRoleRead                   = "read"
	ProjectRoleManage                 = "manage"
	ProjectRoleManageNotification     = "manage-notification"
	ProjectRoleManageWorkerModel      = "manage-worker-model"
	ProjectRoleManageAction           = "manage-action"
	ProjectRoleManageWorkflow         = "manage-workflow"
	ProjectRoleManageWorkflowTemplate = "manage-workflow-template"
	ProjectRoleManageVariableSet      = "manage-variableset"

	// Hatchery Role
	HatcheryRoleSpawn = "start-worker"

	// Region Role
	RegionRoleList    = "list"
	RegionRoleExecute = "execute"
	RegionRoleManage  = "manage"
)

type RBAC struct {
	ID             string              `json:"id" db:"id"`
	Name           string              `json:"name" db:"name" cli:"name"`
	Created        time.Time           `json:"created" db:"created"`
	LastModified   time.Time           `json:"last_modified" db:"last_modified" cli:"last_modified"`
	Global         []RBACGlobal        `json:"global,omitempty" db:"-"`
	Projects       []RBACProject       `json:"projects,omitempty" db:"-"`
	Regions        []RBACRegion        `json:"regions,omitempty" db:"-"`
	Hatcheries     []RBACHatchery      `json:"hatcheries,omitempty" db:"-"`
	Workflows      []RBACWorkflow      `json:"workflows,omitempty" db:"-"`
	VariableSets   []RBACVariableSet   `json:"variablesets,omitempty" db:"-"`
	RegionProjects []RBACRegionProject `json:"region_projects,omitempty" db:"-"`
}

func (rbac *RBAC) IsEmpty() bool {
	return len(rbac.Projects) == 0 && len(rbac.Hatcheries) == 0 && len(rbac.Global) == 0 && len(rbac.Regions) == 0 && len(rbac.VariableSets) == 0 && len(rbac.Workflows) == 0
}

type PermissionSummary struct {
	Global   []string                            `json:"global,omitempty"`
	Regions  map[string][]string                 `json:"regions,omitempty"`
	Projects map[string]PermissionSummaryProject `json:"projects,omitempty"`
}

type PermissionSummaryProject struct {
	Roles        []string            `json:"roles"`
	Workflows    map[string][]string `json:"workflows"`
	VariableSets map[string][]string `json:"variable_sets"`
}

// RBACsToPermissionSummary aggregates a slice of RBAC rules into a PermissionSummary.
// Roles are deduplicated and sorted across all RBAC entries.
// For workflows or variable sets with AllWorkflows/AllVariableSets set to true, the wildcard "*" is used as name.
func RBACsToPermissionSummary(rbs []RBAC) PermissionSummary {
	uniqueAndSort := func(ss []string) []string {
		seen := make(map[string]struct{}, len(ss))
		out := make([]string, 0, len(ss))
		for _, s := range ss {
			if _, ok := seen[s]; !ok {
				seen[s] = struct{}{}
				out = append(out, s)
			}
		}
		sort.Strings(out)
		return out
	}

	// --- Global roles ---
	globalRoles := make([]string, 0)
	for _, rb := range rbs {
		for _, g := range rb.Global {
			globalRoles = append(globalRoles, g.Role)
		}
	}

	// --- Regions ---
	regionRoles := make(map[string][]string)
	for _, rb := range rbs {
		for _, r := range rb.Regions {
			key := r.RegionName
			regionRoles[key] = append(regionRoles[key], r.Role)
		}
	}

	// --- Projects ---
	type projectData struct {
		roles        []string
		workflows    map[string][]string
		variableSets map[string][]string
	}
	projects := make(map[string]*projectData)

	// func to create project perm struct
	ensureProject := func(key string) *projectData {
		if _, ok := projects[key]; !ok {
			projects[key] = &projectData{
				workflows:    make(map[string][]string),
				variableSets: make(map[string][]string),
			}
		}
		return projects[key]
	}

	for _, rb := range rbs {
		for _, p := range rb.Projects {
			for _, key := range p.RBACProjectKeys {
				pd := ensureProject(key)
				pd.roles = append(pd.roles, p.Role)
			}
		}
		for _, wf := range rb.Workflows {
			pd := ensureProject(wf.ProjectKey)
			if wf.AllWorkflows {
				pd.workflows["*"] = append(pd.workflows["*"], wf.Role)
			} else {
				for _, name := range wf.RBACWorkflowsNames {
					pd.workflows[name] = append(pd.workflows[name], wf.Role)
				}
			}
		}
		for _, vs := range rb.VariableSets {
			pd := ensureProject(vs.ProjectKey)
			if vs.AllVariableSets {
				pd.variableSets["*"] = append(pd.variableSets["*"], vs.Role)
			} else {
				for _, name := range vs.RBACVariableSetNames {
					pd.variableSets[name] = append(pd.variableSets[name], vs.Role)
				}
			}
		}
	}

	// --- Build result ---
	summary := PermissionSummary{
		Global:   uniqueAndSort(globalRoles),
		Regions:  make(map[string][]string, len(regionRoles)),
		Projects: make(map[string]PermissionSummaryProject, len(projects)),
	}

	for region, roles := range regionRoles {
		summary.Regions[region] = uniqueAndSort(roles)
	}

	for key, pd := range projects {
		sp := PermissionSummaryProject{
			Roles:        uniqueAndSort(pd.roles),
			Workflows:    make(map[string][]string, len(pd.workflows)),
			VariableSets: make(map[string][]string, len(pd.variableSets)),
		}
		for name, roles := range pd.workflows {
			sp.Workflows[name] = uniqueAndSort(roles)
		}
		for name, roles := range pd.variableSets {
			sp.VariableSets[name] = uniqueAndSort(roles)
		}
		summary.Projects[key] = sp
	}

	return summary
}
