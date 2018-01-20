---
title: "tat2es"
weight: 3
toc: true
prev: "/ecosystem/tatdashing"
next: "/ecosystem/tat2xmpp"

---

Get messages from TAT and send them to ElasticSearch.

## TAT2ES Configuration

```bash
export TAT2ES_LISTEN_PORT="8080"
export TAT2ES_USERNAME_TAT_ENGINE="tat.system.jabber"
export TAT2ES_PRODUCTION=true
export TAT2ES_PASSWORD_TAT_ENGINE="very-long-tat-password-of-tat.system.jabber"
export TAT2ES_URL_TAT_ENGINE="http://tat.your-domain"
export TAT2ES_TOPICS_INDEXES="/Topic/Sub-Topic1:ES_Index1,/Topic/Sub-Topic2:ES_Index2"

# Run TAT2ES
./api -h
```

## Usage

### Building

```bash
mkdir -p $GOPATH/src/github.com/ovh
cd $GOPATH/src/github.com/ovh
git clone git@github.com:ovh/tat-contrib.git
cd tat-contrib/tat2es/api
go build
./api -h
```

### Flags

```bash

./api -h
TAT To ElasticSearch

Usage:
  tat2es [flags]
  tat2es [command]

Available Commands:
  version     Print the version.

Flags:
      --cron-schedule string         Cron Schedule, see https://godoc.org/github.com/robfig/cron (default "@every 3h")
      --host-es string               Host ElasticSearch
      --listen-port string           Tat2ES Listen Port (default "8086")
      --log-level string             Log Level: debug, info or warn
      --messages-limit int           messages-limit is used by MessageCriteria.Limit for requesting TAT (default 1478642112)
      --password-es string           Password ElasticSearch
      --password-tat-engine string   Password Tat Engine
      --port-es string               Port ElasticSearch (default "9200")
      --production                   Production mode
      --timestamp int                from: timestamp unix format (default 1478642112)
      --topics-indexes string        /Topic/Sub-Topic1:ES_Index1,/Topic/Sub-Topic2:ES_Index2
      --url-tat-engine string        URL Tat Engine (default "http://localhost:8080")
      --user-es string               User ElasticSearch
      --username-tat-engine string   Username Tat Engine

Use "tat2es [command] --help" for more information about a command.
```
