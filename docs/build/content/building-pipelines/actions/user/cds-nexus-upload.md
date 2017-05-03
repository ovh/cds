+++
title = "cds-nexus-upload"
chapter = true

[menu.main]
parent = "actions-user"
identifier = "cds-nexus-upload"

+++

Upload file on Nexus

## Parameters

* **artifactId**: Artifact id of the artifact
* **login**: Login for nexus
* **password**: Password for nexus
* **groupId**: Group id of the artifact
* **version**: Version of the artifact. Supports resolving of 'LATEST', 'RELEASE' and snapshot versions ('1.0-SNAPSHOT') too.
* **files**: Regex of files you want to upload
* **packaging**: Packaging type of the artifact
* **url**: Nexus URL
* **repository**: Nexus repository that the artifact is contained in
* **extension**: Extension of the artifact


## Requirements

* **bash**: type:binary Value:bash
* **curl**: type:binary Value:curl


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-nexus-upload.hcl)


