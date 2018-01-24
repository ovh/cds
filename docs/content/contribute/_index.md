+++
title = "Opensource / Contribute"
weight = 7

+++

## Roadmap

See https://github.com/ovh/cds/blob/master/ROADMAP.md

## Contact us

* Benjamin COENEN - [@BnJ25](https://twitter.com/BnJ25)
* François SAMIN - [@francoissamin](https://twitter.com/francoissamin)
* Steven GUIHEUX - [@sguiheux](https://twitter.com/sguiheux)
* Yvonnick ESNAULT - [@yesnault](https://twitter.com/yesnault)

A remark / question / suggestion, feel free to join us on https://gitter.im/ovh-cds/Lobby

All CDS Contributors: https://github.com/ovh/cds/graphs/contributors

## Found a bug?

Feel free to open an issue on https://github.com/ovh/cds/issues

## Documentation

Documentation https://ovh.github.io/cds is generated with Hugo. Source are under https://github.com/ovh/cds/tree/master/docs/content

### Write / Generate / Test documentation

* Download release Hugo v32.4 https://github.com/gohugoio/hugo/releases/tag/v0.32.4 - put hugo binary in your PATH
* Download CDS Binaries: cdsctl, engine, worker from https://github.com/ovh/cds/releases/latest
* Clone CDS repository: git clone https://github.com/ovh/cds.git inside ${CDS_SOURCES}
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


### Golang Development

We use https://github.com/golang/dep to manage CDS Dependencies.

If you have to add or update a CDS Dependency, please follow instructions on https://github.com/golang/dep#adding-a-dependency

And then launch:
```bash
$ dep prune
```
