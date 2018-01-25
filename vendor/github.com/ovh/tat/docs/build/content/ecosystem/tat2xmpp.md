---
title: "tat2xmpp"
weight: 4
toc: true
prev: "/ecosystem/tat2es"

---

## What's for?

tat2xmpp allow you to:

* sync XMPP conference with a Tat topic, from conference to tat, from tat to conference, or both.
* request Tat over XMPP

On your XMPP Client:

```
/tat help
```

will returns:

```
Begin conversation with "tat," or "/tat"

Simple request: "tat, ping"

Request tat:
 "/tat COUNT /Internal/Alerts?tag=NETWORK,label=open"
 "/tat GET /Internal/Alerts?tag=PUBCLOUD-serv,PUBCLOUD-host&label=open"

Request tat and format output:
 "/tat COUNT /Internal/Alerts?tag=NETWORK,label=open format:dateUpdate,username,text"

Default format:dateUpdate,username,text,labels

You can use:
id,text,topic,inReplyOfID,inReplyOfIDRoot,nbLikes,labels,
votersUP,votersDown,nbVotesUP,nbVotesDown,userMentions,
urls,tags,dateCreation,dateUpdate,username,fullname,nbReplies,tatwebuiURL

User tat.system.jabber have to be RO on tat topic for requesting tat.

Get aliases : "/tat aliases", same as "/tat aliases common"
Get aliases with a specific tag : "/tat aliases atag"

Execute an alias : "/tat !myAlias arg1 arg2"

If you add a tat message, with label "common" and text:
"#tatbot #alias #alias:alert #get:/Internal/Alerts?tag=%s&label=%s #format:dateUpdate,text"
you can execute it over XMPP as : "/tat !alert CD open"

For a count request:
"#tatbot #alias #alias:alert.count #count:/Internal/Alerts?tag=%s&label=%s"
you can execute it over XMPP as : "/tat !alert.count CD open"

```

## TAT configuration

```bash

[...]
# TAT 2 XMPP Configuration
exportTAT_TAT2XMPP_USERNAME=tat.system.jabber
exportTAT_TAT2XMPP_URL=http://tat2xmpp.your-domain
exportTAT_TAT2XMPP_KEY=a-key-used-by-tat2xmpp
[...]

# Running TAT Engine
./api
```

## TAT2XMPP Configuration

```bash
export TAT2XMPP_LISTEN_PORT=8080
export TAT2XMPP_HOOK_KEY=a-key-used-by-tat2xmpp
export TAT2XMPP_USERNAME_TAT_ENGINE=tat.system.jabber
export TAT2XMPP_XMPP_BOT_PASSWORD=password-of-bot-user-on-xmpp
export TAT2XMPP_PRODUCTION=true
export TAT2XMPP_PASSWORD_TAT_ENGINE=very-long-tat-password-of-tat.system.jabber
export TAT2XMPP_XMPP_BOT_JID=robot.tat@your-domain
export TAT2XMPP_XMPP_SERVER=your-jabber-server:5222
export TAT2XMPP_URL_TAT_ENGINE=http://tat.your-domain
export TAT2XMPP_URL_TATWEBUI=https://tatwebui.your-domain/standardview/list
export TAT2XMPP_MORE_HELP="TatBot doc: https://ovh.github.io/tat/ecosystem/tat2xmpp"
export TAT2XMPP_ADMIN_TAT2XMPP="usera@jabber.your-domain.net,userb@jabber.your-domain.net,userc@jabber.your-domain.net",

# Running TAT2XMPP
./tat2xmpp
```


## Usage

### Building

```bash
mkdir -p $GOPATH/src/github.com/ovh
cd $GOPATH/src/github.com/ovh
git clone git@github.com:ovh/tat-contrib.git
cd tat-contrib/tat2xmpp/api
go build
./api -h
```

### Flags

```bash

./api -h
Tat2XMPP

Usage:
  tat2xmpp [flags]
  tat2xmpp [command]

Available Commands:
  version     Print the version.

Flags:
      --admin-tat2xmpp string        Admin tat2xmpp admina@jabber.xxx.net,adminb@jabber.xxx.net,
  -c, --configFile string            configuration file
      --hook-key string              Hook Key, for using POST http://<url>/hook endpoint, with Header TAT2XMPPKEY
      --listen-port string           Tat2XMPP Listen Port (default "8088")
      --log-level string             Log Level : debug, info or warn
      --password-tat-engine string   Password Tat Engine
      --production                   Production mode
      --url-tat-engine string        URL Tat Engine (default "http://localhost:8080")
      --url-tatwebui string          TatwebUI base URL
      --username-tat-engine string   Username Tat Engine (default "tat.system.xmpp")
      --xmpp-bot-jid string          XMPP Bot JID (default "tat@localhost")
      --xmpp-bot-password string     XMPP Bot Password
      --xmpp-debug                   XMPP Debug
      --xmpp-delay int               Delay between two sent messages (default 5)
      --xmpp-hello-world string      Sending Hello World message to this jabber id
      --xmpp-insecure-skip-verify    XMPP InsecureSkipVerify (default true)
      --xmpp-notls                   XMPP No TLS (default true)
      --xmpp-server string           XMPP Server
      --xmpp-session                 XMPP Session (default true)
      --xmpp-starttls                XMPP Start TLS

Use "tat2xmpp [command] --help" for more information about a command.

```
