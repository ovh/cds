---
title: "CDN"
weight: 3
---

<b style="color: red">⚠ Do not activate CDN log processing in production yet. It's in active development.
Be sure that config flag 'enableLogProcessing' is set to false</b>

## What's CDN

CDN is a service dedicated to receive and retrieve logs. 

A CDN instance is started with scope Project and Run using token.

CDN is linked to 2 types of storages:

* Buffer Unit: to store step log temporarily during the execution of a job

* Storage Unit: to store complete step logs when it ends

## Use case

Workers and hatcheries communicate with CDN, sending step logs and service log
![CDN_RECEIVE](/images/cdn_receive.png)


CDS UI and CLI communicate with CDN to get entire logs, or stream them
![CDN_GET](/images/cdn_get.png)
