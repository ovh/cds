+++
title = "cds-nexus-upload"

[menu.main]
parent = "actions-user"
identifier = "cds-nexus-upload"

+++

Upload file on Nexus

## Parameters

* **files**: Regex of files you want to upload
* **login**: Login for nexus
* **url**: Nexus URL
* **extension**: Extension of the artifact
* **artifactId**: Artifact id of the artifact
* **version**: Version of the artifact. Supports resolving of 'LATEST', 'RELEASE' and snapshot versions ('1.0-SNAPSHOT') too.
* **packaging**: Packaging type of the artifact
* **password**: Password for nexus
* **repository**: Nexus repository that the artifact is contained in
* **groupId**: Group id of the artifact


## Requirements

* **bash**: type:binary Value:bash
* **curl**: type:binary Value:curl


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-nexus-upload.hcl)


