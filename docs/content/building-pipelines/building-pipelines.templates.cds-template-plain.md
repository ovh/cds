+++
title = "cds-template-plain"
chapter = true

[menu.main]
parent = "templates"
identifier = "cds-template-plain"

+++


This template creates:

- a build pipeline with	two stages: Commit Stage and Packaging Stage
- a deploy pipeline with one stage: Deploy Stage

Commit Stage :

- run git clone
- run make build

Packaging Stage:

- run docker build and docker push

Deploy Stage:

- it's en empty script

Packaging and Deploy are optional.


## Parameters

* **repo**: Your source code repository
* **withPackage**: Do you want a Docker Package?
* **withDeploy**: Do you want an deploy Pipeline?


## More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/templates/cds-template-plain/README.md)

