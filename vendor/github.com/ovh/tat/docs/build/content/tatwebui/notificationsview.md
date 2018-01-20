---
title: "Notifications View"
weight: 3
toc: true
prev: "/tatwebui/standardview"
next: "/tatwebui/monitoringview"

---

When a message contains a mention for someone, a other message is created by Tat Engine in topic
/Private/username/Notifications, where username is the user mentioned.

Example :

* a message "a first notification for @yesnault" in topic /Internal/App
* a other message "and a second @yesnault in topic /Internal/App" in topic /Internal/App

see result in screenshot below for these two messages in topic /Private/yesnault/Notifications

## Screenshot

![Notifications View](/imgs/tatwebui-notifications-view.png?width=80%)

## Configuration
In plugin.tpl.json file, add this line :

```
"tatwebui-plugin-notificationsview": "git+https://github.com/ovh/tatwebui-plugin-notificationsview.git"
```

## Source
[https://github.com/ovh/tatwebui-plugin-notificationsview](https://github.com/ovh/tatwebui-plugin-notificationsview)
