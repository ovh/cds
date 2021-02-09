---
title: "CDN"
weight: 3
---

## What's CDN
CDN is a service dedicated to receive and store CDS's logs. In a near future it will also be able to manage artifacts and cache used by your jobs. 

CDN stores the list of all known log or artifact items in a Postgres database and communicate with storage backends to store the content of this items.
This backends are call units and there are two types of units in CDN:

* Buffer Unit: to store incoming job's logs and artifact, this units are designed to be fast for read/write operations but with a limited capacity.

* Storage Unit: to store complete job's logs and artifact.

## Configuration
Like any other CDS service, CDN requires to be authenticated with a consumer. The required scope are Service, Worker and RunExecution.
You must have at least one storage unit, one file buffer and one log buffer to be able to run CDN.

## Supported units
* Buffer (type: log): Redis.
* Buffer (type: file): Local.
* Storage: Local, Swift, Webdav, CDS (cds unit is used for migration purpose and will be removed in future release).


## Use case

Workers and hatcheries communicate with CDN, sending step logs and service log
![CDN_RECEIVE](/images/cdn_receive.png)

CDS UI and CLI communicate with CDN to get entire logs, or stream them
![CDN_GET](/images/cdn_get.png)
