# CDS: Continuous Delivery Service

[![Join the chat at https://gitter.im/ovh-cds/Lobby](https://badges.gitter.im/ovh-cds/Lobby.svg)](https://gitter.im/ovh-cds/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/ovh/cds)](https://goreportcard.com/report/github.com/ovh/cds)
[![Coverage Status](https://coveralls.io/repos/github/ovh/cds/badge.svg?branch=master)](https://coveralls.io/github/ovh/cds?branch=master)
[![GoDoc](https://godoc.org/github.com/ovh/cds/sdk?status.svg)](https://godoc.org/github.com/ovh/cds/sdk)

<img align="right" src="https://raw.githubusercontent.com/ovh/cds/master/logo-background.png" width="25%">

CDS is a pipeline based Continuous Delivery Service written in Go(lang).

**This project is under active development**

**[Documentation](https://ovh.github.io/cds/)**


## Super intuitive UI
CDS provides a super intuitive UI that allows you to build complex workflows, run them and dig into the logs when needed.

<img src="./docs/static/images/capture-start.gif" alt="CDS Demonstration" width="80%">


## The most powerful Command Line for a CI/CD Platform

cdsctl is the CDS Command Line - you can script everything with it, cdsctl also provide some cool commands as `cdsctl shell` to browse your projects and workflows without the need to open a browser.

[![cdsctl shell](https://asciinema.org/a/fTFpJ5uqClJ0Oq2EsiejGSeBk.svg)](https://asciinema.org/a/fTFpJ5uqClJ0Oq2EsiejGSeBk)

[See all cdsctl commands](https://ovh.github.io/cds/cli/cdsctl/#see-also)


## Want a try?

Docker-Compose or Helm are your friends, see [Ready To Run Tutorials](https://ovh.github.io/cds/hosting/ready-to-run/)

## FAQ

### Why CDS? Discover the Origins

- [Self-Service](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#1-self-service)
- [Horizontal Scalability](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#2-horizontal-scalability)
- [High Availability](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#3-high-availability)
- [Pipeline Reutilisability](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#4-pipeline-reutilisability)
- [Rest API](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#5-rest-api)
- [Customizable](https://ovh.github.io/cds/gettingstarted/concepts/why_cds/#6-customizable)

### What is a CDS workflow?

Most of the CI/CD Tools play with jobs inside a pipeline. CDS introduce a new concept named `CDS Workflows`.
A [CDS Workflow](https://ovh.github.io/cds/gettingstarted/concepts/workflow/) allows you to chain pipelines with triggers.
A [pipeline](https://ovh.github.io/cds/gettingstarted/concepts/pipeline/) is structured in sequential [stages](https://ovh.github.io/cds/gettingstarted/concepts/stage/) containing one or multiple concurrent [jobs](https://ovh.github.io/cds/gettingstarted/concepts/job/).


### Can I use it in production?

Yes! CDS is used in production since 3y @OVH and launch more than 7M CDS workers per year. You can install the official release available on https://github.com/ovh/cds/releases

CDS provides everything needed to monitor and measure production activity (logs, metrics, monitoring)

### How to backup?

All data are stored in the database - nothing on filesystem. Just backup your database regularly and you will be safe.

### Need some help?

Core Team is available on [Gitter](https://gitter.im/ovh-cds/Lobby)


## License

[3-clause BSD](./LICENCE)
