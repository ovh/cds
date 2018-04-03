+++
title = "cds-split-download"

+++

Download Artifacts which were uploaded using CDS_SplitUploadArtifact action.

## Parameters

* **destinationFile**: Destination File to save the file. (Mandatory). (like dist.tar.gz)
* **pattern**: Prefix pattern to identify files to be downloaded. (Mandatory). (like bigfile-*)
* **prefixHandle**: Prefix of the Artifacts uploaded. (Mandatory).
* **tag**: Tag to identify uploaded artifacts. Default tag value is CDS run version number. (Optional)


## Requirements

* **bash**: type: binary Value: bash


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-split-download.yml)


