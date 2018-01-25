---
title: "tatdashing"
weight: 4
toc: true
prev: "/ecosystem/mail2tat"
next: "/ecosystem/tat2es"

---

## Overview

Each 30seconds, tatDashing computes dashboard. This uService gets information on tat and "relabel" messages on your dashing topic.

## Build And Run

### Configuration

```bash

export TATDASHING_PRODUCTION=true
export TATDASHING_LISTEN_PORT=8080
export TATDASHING_PASSWORD_TAT_ENGINE="very-long-tat-password-of-tat.system.dashing"
export TATDASHING_URL_TAT_ENGINE="http://tat.your-domain"
export TATDASHING_USERNAME_TAT_ENGINE="tat.system.dashing"

# Run TatDashing
./api -h
```

### Build

```bash
mkdir -p $GOPATH/src/github.com/ovh
cd $GOPATH/src/github.com/ovh
git clone git@github.com:ovh/tat-contrib.git
cd tat-contrib/tatdashing/api
go build
./api -h
```

### Flags

```bash
$ ./api -h

Tat Dashing

Usage:
  tatdashing [flags]

Flags:
      --listen-port string           Listen Port (default "8085")
      --log-level string             Log Level : debug, info or warn
      --password-tat-engine string   Password Tat Engine
      --production                   Production mode
      --url-tat-engine string        URL Tat Engine (default "http://localhost:8080")
      --username-tat-engine string   Username Tat Engine (default "tat.system.dashing")
```

## Usage

To use it, you have to :

* Add user tat.system.dashing read write on your dashing topic
* Add user tat.system.dashing read only on topics where you have your data. For example, in examples below, tat.system.dashing is Read Only on topics /Internal/Alerts and /Internal/PullRequests
* Create initial root message with like "#monitoring #item:Alerts", see full documentation about that on https://ovh.github.io/tat/tatwebui/dashingview/
 * Add labels on this root message, like "order:12", "url
 * Add reply for each label you want to rewrite

### Compute color and bg-color

Example for compute label "color":

* `#label:color:08f1f4:0:10,ce352c` : if value is >= 0 and value <= 10, color will be #08f1f4, else color will be #ce352c
* `#label:color:08f1f4:0:10,ce352c:11:20,fa6800` : if value is >= 0 and value <= 10, color will be #08f1f4, else if value is >=11 and value <= 20, color will be #ce352c, else  color will be #fa6800

### Compute value(s)

This reply : #TatDashing #label:0:value #value:0:/Internal/Alerts?tag=CD&notLabel=done&onlyMsgRoot=true will :

* compute label "value" by
* count messages from /Internal/Alerts with tag=CD and notLabel=done and onlyMsgRoot=true to avoid counting exclude alert replay.


### Compute value(s) with label value

This reply : #TatDashing #label:0:value #valuelabel:0:qos/Internal/Alerts?tag=CD will :</p>

* compute "value" with
* value of the label "qos:xxx" from /Internal/Alerts with tag=CD.

## Examples

Display alerts from /Internal/Alerts with tag=CD and notLabel=done. Don't forget onlyMsgRoot for exclude alert replay

![View Alerts](/imgs/tatdashing-alerts-view.png?width=50%)

Labels color, bg-color and value are computed by TatDashing uService.

![Details Alerts](/imgs/tatdashing-alerts-details.png?width=50%)

Display Opened and Approved Pull Requests for projects TEXTANDTAGS, CDS and CD,

![View PullRequests](/imgs/tatdashing-pullRequest-view.png?width=50%)

Labels color, bg-color and value are computed by TatDashing uService.

![Details PullRequests](/imgs/tatdashing-pullRequest-details.png?width=50%)

A graph with Chartist (see. https://gionkunz.github.io/chartist-js/examples.html )

![View Advanced Graph](/imgs/tatdashing-complex-view.png?width=50%)

Label widget-data-series is computed by TatDashing uService. value:0 is first point, value:1 is second point, etc...

![Details Advanced Graph](/imgs/tatdashing-complex-details.png?width=50%)
