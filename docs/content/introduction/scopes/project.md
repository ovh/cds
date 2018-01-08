+++
title = "Project"
weight = 2

[menu.main]
parent = "scopes"
identifier = "scopes.project"

+++

## Project

A project contains applications, pipelines and environments.

A project is the first level of permissions management. Any CDS application has to be created inside a project.

The project key has to be unique among all projects in CDS.

At creation, a project has to have at least one group with edition permissions on it. It is possible to use either an existing group or create a new one.

If the provided group does not exist, a default group will be created with edition permissions on project and the group creator will be automatically created to that group.

