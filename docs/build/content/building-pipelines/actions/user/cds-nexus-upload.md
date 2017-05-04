+++
title = "cds-nexus-upload"

[menu.main]
parent = "actions-user"
identifier = "cds-nexus-upload"

+++

Upload file on Nexus

## Parameters

* **url**: Nexus URL
* **repository**: Nexus repository that the artifact is contained in
* **groupId**: Group id of the artifact
* **version**: Version of the artifact. Supports resolving of 'LATEST', 'RELEASE' and snapshot versions ('1.0-SNAPSHOT') too.
* **packaging**: Packaging type of the artifact
* **login**: Login for nexus
* **password**: Password for nexus
* **extension**: Extension of the artifact
* **artifactId**: Artifact id of the artifact
* **files**: Regex of files you want to upload


## Requirements

* **bash**: type:binary Value:bash
* **curl**: type:binary Value:curl


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-nexus-upload.hcl)


