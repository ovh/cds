# CDS Hubot-XMPP

This µservice:
- run [Hubot](https://hubot.github.com/) with an adapter xmpp
- expose `/cds/notifications/xmpp`: allow you to POST CDS Event to XMPP users and rooms
- expose `/health`: answer HTTP Code 200

## Example

```bash

$ curl \
    --header "Content-Type: application/json" \
    --request POST \
    --data "{ recipients: [ 'a-user@localhost.local', 'a-room@localhost.local'], subject: 'title', body: 'message' }" \
    http://127.0.0.1:8080/cds/notifications/xmpp

```

`--data` is a [EventNotif](https://godoc.org/github.com/ovh/cds/sdk#EventNotif).

You can use [cds2http](https://github.com/ovh/cds/tree/master/contrib/uservices/cds2http) µservice to send CDS Event to CDS Hubot-xmpp.


## build and run it

```bash
$ docker build -t cdsbot-image .
$ docker run \
-e HUBOT_XMPP_ROOMS=your-room@localhost.local  \
-e HUBOT_XMPP_HOST=xmpp_server.localhost.local  \
-e HUBOT_XMPP_PASSWORD=your-password  \
-e HUBOT_XMPP_PORT=5222  \
-e HUBOT_XMPP_USERNAME=robot.cds@localhost.local  \
-e HUBOT_XMPP_DEFAULT_DOMAIN=@localhost.local \
-it cdsbot-image
```
