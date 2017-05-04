+++
title = "cds-template-only-git-clone-job"
chapter = true

[menu.main]
parent = "templates"
identifier = "cds-template-only-git-clone-job"

+++


This template creates:

- a build pipeline with	one stage, containing one job
- job contains 2 steps: GitClone and a empty script.

Pipeline name contains Application name.
If you want to make a reusable pipeline, please consider updating this name after creating application.


## Parameters

* **repo**: Your source code repository


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/templates/cds-template-only-git-clone-job/README.md)

