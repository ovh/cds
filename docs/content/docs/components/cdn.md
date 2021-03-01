---
title: "CDN"
weight: 3
---

## What's CDN
CDN is a service dedicated to receive and store CDS's logs. In a near future, it will also be able to manage artifacts and cache used by your jobs. 

CDN stores the list of all known log or artifact items in a Postgres database and communicates with storage backends to store the contents of those items.
These backends are call units and there are two types of units in CDN:

* Buffer unit: To store logs and artifacts of incoming jobs, these units are designed to be fast for read / write operations, but with limited capacity.

* Storage Unit: to store complete job's logs and artifact.

When logs or file are received by CDN, it will first store these items in its buffer. Then, when the item is fully received, it will be moved to one of the configured storage units.
If the CDN service is configured with multiple storage units, each unit periodically checks for missing items and synchronizes these items from other units.


## Configuration
Like any other CDS service, CDN requires to be authenticated with a consumer. The required scopes are Service, Worker and RunExecution.

You must have at least one storage unit, one file buffer and one log buffer to be able to run CDN.

## Supported units
* Buffer (type: log): Redis.
* Buffer (type: file): Local.
* Storage: Local, Swift, S3, Webdav, CDS (cds unit is used for migration purpose and will be removed in future release).


## Use case

Workers and hatcheries communicate with CDN, sending step logs and service log.

![CDN_RECEIVE](/images/cdn_logs_receive.png?width=600px)

CDS UI and CLI communicate with CDN to get entire logs, or stream them.

![CDN_GET](/images/cdn_logs_get.png?width=600px)
