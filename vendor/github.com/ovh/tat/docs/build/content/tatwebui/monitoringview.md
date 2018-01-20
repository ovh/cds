---
title: "Monitoring View"
weight: 4
toc: true
prev: "/tatwebui/notificationsview"
next: "/tatwebui/pastatview"

---


## Screenshot

![Monitoring View](/imgs/tatwebui-monitoring-view.png?width=80%)

## Using

### Send to Tat Engine

The screenshot above was created with these messages:

```bash

tatcli topic truncate /Private/yesnault/Monitoring --force
for i in {1..99}; do
  for j in {1..20}; do
     MTYPE="UP";
     COLOR="#6C6";
     if [ ${j} -eq 7 ]; then MTYPE="AL"; COLOR="d9534f"; fi;
     tatcli msg add /Private/yesnault/Monitoring "#monitoring #myService #item:myItem${i}${j}" --label="$COLOR;$MTYPE"
  done
done

```

### Production Way with al2tat

Send a monitoring message to al2tat microservice on path /monitoring.
See https://github.com/ovh/al2tat


## Configuration

In plugin.tpl.json file, add this line :

```
"tatwebui-plugin-monitoringview": "git+https://github.com/ovh/tatwebui-plugin-monitoringview.git"
```

## Source
[https://github.com/ovh/tatwebui-plugin-monitoringview](https://github.com/ovh/tatwebui-plugin-monitoringview)
