+++
title = "Write documentation"
weight = 2

+++

Documentation https://ovh.github.io/cds is generated with Hugo. Source are under https://github.com/ovh/cds/tree/master/docs/content

Write / Generate / Test documentation:

* Download release Hugo v32.4 https://github.com/gohugoio/hugo/releases/tag/v0.32.4 - put hugo binary in your PATH
* Download CDS Binaries: cdsctl, engine, worker from https://github.com/ovh/cds/releases/latest
* Clone CDS repository: `git clone https://github.com/ovh/cds.git` inside ${CDS_SOURCES}
* Generate documentation with Hugo

```bash
cd ${CDS_SOURCES}/docs/content/cli
rm -rf cdsctl engine worker;
cd ${CDS_SOURCES}
GEN_PATH=${CDS_SOURCES}/docs/content/cli make doc 
cd ${CDS_SOURCES}/docs
hugo server
```
* go to http://localhost:1313/cds/
