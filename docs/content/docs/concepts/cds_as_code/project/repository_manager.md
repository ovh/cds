---
title: "Repository Manager"
weight: 2
---

# Description

A repository manager is needed by your CDS project to link repositories.

A project can be linked to one or more repository manager:

* [github]({{< relref "/docs/integrations/github/_index.md" >}})
* [bitbucket server]({{< relref "/docs/integrations/bitbucket.md" >}})
* [gitlab]({{< relref "/docs/integrations/gitlab/_index.md" >}})

# Permission

To be able to manage repository manager you will need the permission `manage` on your project

# Add a repository manager with CLI

```
cdsctl experimental project vcs import <PROJECT-KEY> <vcs_file.yaml>
```
* `PROJECT-KEY`: The project key

[Full CLI documentation here]({{< relref "/docs/components/cdsctl/project/vcs/_index.md" >}})