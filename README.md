# CDS: Continuous Delivery Service

[![Go Report Card](https://goreportcard.com/badge/github.com/ovh/cds)](https://goreportcard.com/report/github.com/ovh/cds)
[![GoDoc](https://godoc.org/github.com/ovh/cds/sdk/cdsclient?status.svg)](https://godoc.org/github.com/ovh/cds/sdk/cdsclient)

<img align="right" src="https://raw.githubusercontent.com/ovh/cds/master/logo-background.png" width="25%">

CDS is an Enterprise-Grade Continuous Delivery & DevOps Automation Platform written in Go(lang).

**This project is under active development**

**[Documentation](https://ovh.github.io/cds/)**


## Intuitive UI
CDS provides an intuitive UI that allows you to build complex workflows, run them and dig into the logs when needed.

<p align="center">
  <kbd>
    <img src="./docs/static/images/capture-start.gif" alt="create and run workflow with CDS ui" title="create and run workflow with CDS ui">
  </kbd>
  <i>Create and run workflow with CDS ui.</i>
</p>

## The most powerful Command Line for a CI/CD Platform

cdsctl is the CDS Command Line - you can script everything with it, cdsctl also provide some cool commands such as `cdsctl shell` to browse your projects and workflows without the need to open a browser.

[See all cdsctl commands](https://ovh.github.io/cds/docs/components/cdsctl/)

<p align="center">
  <img src="./docs/static/images/init_template_as_code.gif" alt="create workflow as code with CDS command line" title="create workflow as code with CDS command line">
  <i>Create workflow as code with CDS command line.</i>
</p>

## Want a try?

Docker-Compose is your friend, see [Ready To Run Tutorials](https://ovh.github.io/cds/hosting/ready-to-run/)

## Blog posts and talks

-	CDS Introduction: https://www.ovh.com/fr/blog/how-does-ovh-manage-the-ci-cd-at-scale/
-	DataBuzzWord Podcast (French) : https://www.ovh.com/fr/blog/understanding-ci-cd-for-big-data-and-machine-learning/
- Continuous Delivery and Deployment Workflows with CDS: https://www.ovh.com/fr/blog/continuous-delivery-and-deployment-workflows-with-cds/
- Talk at conference [Breizhcamp](https://www.breizhcamp.org) to introduce CDS (French): https://www.youtube.com/watch?v=JUzEQuOehv4

## FAQ

### Why CDS? Discover the Origins

- [Self-Service](https://ovh.github.io/cds/about/why_cds/#1-self-service)
- [Horizontal Scalability](https://ovh.github.io/cds/about/why_cds/#2-horizontal-scalability)
- [High Availability](https://ovh.github.io/cds/about/why_cds/#3-high-availability)
- [Pipeline Reutilisability](https://ovh.github.io/cds/about/why_cds/#4-pipeline-reutilisability)
- [Rest API](https://ovh.github.io/cds/about/why_cds/#5-rest-api)
- [Customizable](https://ovh.github.io/cds/about/why_cds/#6-customizable)

### What is a CDS workflow?

Most of the CI/CD Tools play with jobs inside a pipeline. CDS introduce a new concept named `CDS Workflows`.
A [CDS Workflow](https://ovh.github.io/cds/docs/concepts/workflow/) allows you to chain pipelines with triggers.
A [pipeline](https://ovh.github.io/cds/docs/concepts/pipeline/) is structured in sequential [stages](https://ovh.github.io/cds/docs/concepts/pipeline/#stages) containing one or multiple concurrent [jobs](https://ovh.github.io/cds/docs/concepts/job/).


### Can I use it in production?

Yes! CDS is used in production since 2015 @OVH and it launches more than 7M CDS workers per year. You can install the official release available on https://github.com/ovh/cds/releases

CDS provides everything needed to monitor and measure production activity (logs, metrics, monitoring)

### How to backup?

All data are stored in the database - nothing on filesystem. Just backup your database regularly and you will be safe.

### Need some help?

Core Team is available on https://github.com/ovh/cds/discussions

### Comparison Matrix

See https://ovh.github.io/cds/about/matrix/

## License

[3-clause BSD](./LICENSE)
