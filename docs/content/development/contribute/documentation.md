---
title: "Write documentation"
weight: 2
card: 
  name: contribute
---

Documentation https://ovh.github.io/cds is generated with Hugo. Source are under https://github.com/ovh/cds/tree/master/docs/content

Write / Generate / Test documentation:

* Download release Hugo **Version Extended** v0.54.0 https://github.com/gohugoio/hugo/releases/tag/v0.54.0 - put hugo binary in your PATH
* Download CDS Binaries: cdsctl, engine, worker from https://github.com/ovh/cds/releases/latest
* Clone CDS repository: `git clone https://github.com/ovh/cds.git` inside ${CDS_SOURCES}
* Generate documentation with Hugo

```bash
cd ${CDS_SOURCES}
make install # to recompile all CDS binaries
GEN_PATH=${CDS_SOURCES}/docs/content/docs/components make doc 
cd ${CDS_SOURCES}/docs
hugo server
```
* go to http://localhost:1313/
