---
title: "Monitoring a process with al2tat"
weight: 32
prev: "/tatnhack"
next: "/tatnhack/visual-feedback"
toc: true

---

Here, a script for monitoring process and send alert to al2tat.

Script checkProcess.sh:

```bash
#!/bin/bash

NAME="$1"
HOSTNAME=`hostname`
TAT_USER="tat.system.your.user"
TAT_PASSWORD="YouTatVeryLongPassword"
TAT_TOPIC="/Internal/Alerts"

SERVICE="YourService"

if [[ "x${NAME}" == "x" ]]; then
    echo "invalid usage, ./checkProcess.sh <processName>";
    exit 1;
fi;

pgrep -l ${NAME} > /dev/null 2>&1

if [[ $? -ne 0 ]]; then
    curl -XPOST  \
    -H "Content-Type: application/json" \
    -H "Tat_username: ${TAT_USER}" \
    -H "Tat_password: ${TAT_PASSWORD}" \
    -H "Tat_topic: ${TAT_TOPIC}" \
    -d '{ "alert" : "AL", "nbAlert" : 1, "service" : "'${SERVICE}'", "summary" : "'${NAME}' is down on '${HOSTNAME}'" }' https://<url2tat>/alarm/sync
fi;
```

Crontab

```
*/5 * * * * checkProcess.sh yourProcess > /dev/null 2>&1
```
