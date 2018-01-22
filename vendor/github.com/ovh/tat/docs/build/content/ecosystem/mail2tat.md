---
title: "mail2tat"
weight: 2
toc: true
prev: "/ecosystem/al2tat"
next: "/ecosystem/tatdashing"

---

Create a message on Tat by sending a mail. Mail2tat check an imap account
and create a message for each mail received.

## Tat Configuration

* Add Read Write to user tat.system.mail on your destination topic
* Check option : "System User can force dateCreation of message ?"

## Simple usage

Send a mail to tat@your-domain with
```
subjet : <topicName>
Body : text of message
```
Example of subject : `/Internal/YourTopic`

## Thread on Tat

Send a mail to tat@your-domain with :
```
subjet : <topicName>,<idGroup>
Body: text of message
```

Example of subject: `/Internal/YourTopic,something`

## FAQ
Time between sending mail and post on tat ? Each minute : check mail and send on tat.
Restriction on From ? Yes, see arg  : only @your-domain. All mail received from another domain are not send on tat.

## Usage

### Building

```bash
mkdir -p $GOPATH/src/github.com/ovh
cd $GOPATH/src/github.com/ovh
git clone git@github.com:ovh/tat-contrib.git
cd tat-contrib/mail2tat/api
go build
./api -h
```

### Flags

```
./api -h
MAIL2TAT - Mail to Tat

Usage:
  mail2tat [flags]

Flags:
      --activate-cron                Activate Cron (default true)
      --imap-host string             IMAP Host
      --imap-password string         IMAP Password
      --imap-username string         IMAP Username
      --listen-port string           RunKPI Listen Port (default "8084")
      --log-level string             Log Level : debug, info or warn
      --password-tat-engine string   Password Tat Engine
      --production                   Production mode
      --url-tat-engine string        URL Tat Engine (default "http://localhost:8080")
      --username-tat-engine string   Username Tat Engine
```
