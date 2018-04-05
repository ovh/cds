+++
title = "cds-split-upload"

+++

Split and Upload Artifact with size greater than 500 MB.

## Parameters

* **numericSuffixes**: Use numeric suffixes instead of alphabetic. Default suffix is set to be Numeric. (Optional)
* **prefixHandle**: Prefix to be added to destination files after split. (Mandatory)
* **sourceFile**: Artifact File to be split and uploaded. (Mandatory).
* **splitSize**: Size of each split files. Default size is 200MB. (Optional)
* **tag**: Tag to identify uploaded artifacts. Default tag value is CDS run version number. (Optional)


## Requirements

* **bash**: type: binary Value: bash
* **split**: type: binary Value: split


More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/actions/cds-split-upload.yml)


