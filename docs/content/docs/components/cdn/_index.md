---
title: "CDN"
weight: 3
---

## What's CDN

CDN is a service dedicated to receice and retrieve logs. It replaces the old system that stored logs in the CDS database

A CDN instance is started with scope Project and Run using token.

CDN is linked to 2 types of storages:

* Buffer Unit: to store step log temporarily during the execution

* Storage Unit: to store complete step logs when it ends

## Use case

Workers and hatcheries communicate with CDN, sending step logs and service log
![CDN_RECEIVE](/images/cdn_receive.png#banner)


CDS UI and CLI communicate with CDN to get entire logs, or stream them
![CDN_GET](/images/cdn_get.png#banner)

