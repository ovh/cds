+++
title = "Concepts"
weight = 2

[menu.main]
parent = "introduction"
identifier = "concepts"

+++

![Concepts](/images/concepts_prj.png)

## Project
A project contains applications, pipelines and environments.

A project is the first level of permissions management. Any CDS application has to be created inside a project.

The project key has to be unique among all projects in CDS.

At creation, a project has to have at least one group with edition permissions on it. It is possible to use either an existing group or create a new one.

If the provided group does not exist, a default group will be created with edition permissions on project and the group creator will be automatically created to that group.

## Application

An application represents a real world production unit. An application lives inside a project, has variables and can be attached to:

* [Pipelines]({{< relref "introduction.concepts.pipeline.md" >}})
* Environments

## Environment

An environment is created inside a project and can be used by all the applications of that project.
