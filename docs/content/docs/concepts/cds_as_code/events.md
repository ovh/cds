---
title: "Events"
weight: 6
card:
  name: cds_as_code
  weight: 5
---


# Description

Each action on CDS triggers an event.

# Analysis event

```go
type AnalysisEvent struct {
	ProjectEventV2
	VCSName    string `json:"vcs_name"`
	Repository string `json:"repository"`
	Status     string `json:"status"`
	UserID     string `json:"user_id,omitempty"`
	Username   string `json:"username,omitempty"`
}

type ProjectEventV2 struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	ProjectKey string          `json:"project_key"`
	Payload    json.RawMessage `json:"payload"`
}
```

Status:

* `AnalysisStart`
* `AnalysisDone`

# AsCode entity event

* `EntityCreated`
* `EntityUpdated`
* `EntityDeleted`

# Hachery event

* `HatcheryCreated`
* `HatcheryUpdated`
* `HatcheryDeleted`

# Integration model event

* `IntegrationModelCreated`
* `IntegrationModelUpdated`
* `IntegrationModelDeleted`

# Integration event	

* `IntegrationCreated`
* `IntegrationUpdated`
* `IntegrationDeleted`

# Notification event

* `NotificationCreated`
* `NotificationUpdated`
* `NotificationDeleted`

# Organization event

* `OrganizationCreated`
* `OrganizationDeleted`

# Permission event

* `PermissionCreated`
* `PermissionUpdated`
* `PermissionDeleted`

# Plugin event

* `PluginCreated`
* `PluginUpdated`
* `PluginDeleted`

# Project event

* `ProjectCreated`
* `ProjectUpdated`
* `ProjectDeleted`

# Region event

* `RegionCreated`
* `RegionDeleted`

# Repository event

* `RepositoryCreated`
* `RepositoryDeleted`

# User event

* `UserCreated`
* `UserUpdated`
* `UserDeleted`

# User gpg event

* `UserGPGKeyCreated`
* `UserGPGKeyDeleted`

# VariableSet event

* `VariableSetCreated`
* `VariableSetDeleted`

# VariableSet item event

* `VariableSetItemCreated`
* `VariableSetItemUpdated`
* `VariableSetItemDeleted`

# Workflow Run event

* `RunCrafted`
* `RunBuilding`
* `RunEnded`
* `RunRestartFailedJob`

# Workflow Run Job event

* `RunJobEnqueued`
* `RunJobScheduled`
* `RunJobBuilding`
* `RunJobManualTriggered`
* `RunJobRunResultAdded`
* `RunJobRunResultUpdated`
* `RunJobEnded`
