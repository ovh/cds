+++
title = "cds-docker-package"

[menu.main]
parent = "actions-user"
identifier = "cds-docker-package"

+++

Build a Docker image and push it to a docker repository

## Parameters

* **dockerOpts**: Docker options, you can add `--no-cache --pull` for example
* **dockerRegistry**: The Docker registry to push the image to.
* **dockerRegistryPassword**: Docker Registry Password. Enter password to connect on your docker registry.
* **dockerRegistryUsername**: Docker Registry Username. Enter username to connect on your docker registry.
* **dockerfileDirectory**: Directory which contains your Dockerfile.
* **imageName**: Name of your docker image, without tag.
* **imageTag**: The Docker image tag. {{.cds.version}} can be a good tag value. You can use multiple tags. E.g., firsttag,secondtag,{{.cds.version}},latest


## Requirements

* **bash**: type: binary Value: bash
* **docker**: type: binary Value: docker


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-docker-package.hcl)


