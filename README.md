# CDS: Continuous Delivery Service

[![Join the chat at https://gitter.im/ovh-cds/Lobby](https://badges.gitter.im/ovh-cds/Lobby.svg)](https://gitter.im/ovh-cds/Lobby?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Go Report Card](https://goreportcard.com/badge/github.com/ovh/cds)](https://goreportcard.com/report/github.com/ovh/cds)
[![Coverage Status](https://coveralls.io/repos/github/ovh/cds/badge.svg?branch=master)](https://coveralls.io/github/ovh/cds?branch=master)

<img align="right" src="https://raw.githubusercontent.com/ovh/cds/master/logo-background.png" width="25%">

CDS is a pipeline based Continuous Delivery Service written in Go(lang).

**This project is under active development**

## Documentation

Documentation is available [here](https://ovh.github.io/cds/)

## Overview

CDS is composed of several components

### Engine

The core component of CDS: [read more](/engine/README.md)

### WebUI

CDS Web UI: [read more](ui/README.md)

### CLI

CDS Command line interface: [read more](https://ovh.github.io/cds/cli/cdsctl/)

### Worker

In CDS, a worker is an agent executing actions pushed in queue by CDS engine: [read more](https://ovh.github.io/cds/worker/)

### Hatchery

In CDS, a hatchery is an agent which spawn workers: [read more](https://ovh.github.io/cds/hatchery/)

### Contrib

Actions, Plugins, uServices are under : [read more](contrib)

### SDK

A Go(lang) SDK is available at github.com/ovh/cds/sdk. It provides helper functions for all API handlers, with embedded authentification mechanism.

[![GoDoc](https://godoc.org/github.com/ovh/cds/sdk?status.svg)](https://godoc.org/github.com/ovh/cds/sdk)

## Links

* OVH home (us): https://www.ovh.com/us/
* OVH home (fr): https://www.ovh.com/fr/
* OVH community: https://community.ovh.com/c/open-source/continuous-delivery-service

## License

3-clause BSD
