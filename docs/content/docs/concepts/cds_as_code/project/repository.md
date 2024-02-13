---
title: "Repository"
weight: 3
---

# Description

A repository represents a link between your CDS project and a git repository. 

A repository is analyzed by CDS on each push event in order to detect cds files that represent [CDS as code entities](./../entities/).
During this analysis, CDS will retrieve the user and create/update found entities regarding his [permissions](./../rbac/).

# Permission

To be able to manage repositories you will need the permission `manage` on your project

# Add a repository using CLI

```
cdsctl experimental project repository add <PROJECT-KEY> <VCS-NAME> <REPOSITORY-NAME> 
```
* `PROJECT-KEY`: The project key
* `VCS-NAME`: The vcs name in which the repository is located.
* `REPOSITORY-NAME`: The repository name (<owner>/<repo_name>) to link

[Full CLI documentation here]({{< relref "/docs/components/cdsctl/experimental/project/repository/_index.md" >}})

# Add a repository using UI

Comming soon