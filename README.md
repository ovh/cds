# CDS: Continuous Delivery Service

<img align="right" src="https://raw.githubusercontent.com/ovh/cds/master/logo-background.png">

CDS is a pipeline based Continuous Delivery Service written in Go(lang).

**/!\ This project is under active development.**

## Documentation

Documentation is available [here](/doc/overview/introduction.md)

## Overview

CDS is composend of several components/

### API

The core component of CDS: https://github.com/ovh/cds/tree/master/engine/api

### WebUI

CDS Web UI: https://github.com/ovh/cds/tree/master/ui

### CLI

CDS Command line interface: https://github.com/ovh/cds/tree/master/cli/cds

### Worker

In CDS, a worker is an agent executing actions pushed in queue by CDS engine: [read more](/doc/overview/worker.md)

### Hatchery

In CDS, a hatchery is an agent which spawn workers: [read more](/doc/overview/hatchery.md)

### Contrib

Actions, Plugins, Templates, uServices are under : https://github.com/ovh/cds/tree/master/contrib

### SDK

A Go SDK is available at github.com/ovh/cds/sdk. It provide helper functions for all API handlers, with embedded authentification mechanism.

## Links

-OVH home (us): https://www.ovh.com/us/
-OVH home (fr): https://www.ovh.com/fr/
-OVH community: https://community.ovh.com/c/open-source/continuous-delivery-service

## License

3-clause BSD
