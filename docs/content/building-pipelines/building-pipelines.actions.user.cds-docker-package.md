+++
title = "cds-docker-package"

[menu.main]
parent = "actions-user"
identifier = "cds-docker-package"

+++

Build image and push it to docker repository

## Parameters

* **imageTag**: Tag of your docker image.
Enter mytag for build image myregistry/myimage:mytag. {{.cds.version}} is a good tag from CDS.
You can use many tags: firstTag,SecondTag
Example : {{.cds.version}},latest
* **dockerfileDirectory**: Directory which contains your Dockerfile.
* **dockerOpts**: Docker options, Enter --no-cache --pull if you want for example
* **dockerRegistry**: Docker Registry. Enter myregistry for build image myregistry/myimage:mytag
* **imageName**: Name of your docker image, without tag. Enter myimage for build image myregistry/myimage:mytag


## Requirements

* **docker**: type:binary Value:docker
* **bash**: type:binary Value:bash


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-docker-package.hcl)


