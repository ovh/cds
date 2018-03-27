+++
title = "Service Requirement PostgreSQL"
weight = 2

+++

#### Add the service requirement

Requirement Service

* Type: `service`
* Hostname: `mypg`. This will be the service hostname
* Docker Image: `postgres:9.5.3`. This is the name of docker image to link to current job
* Options:

```
POSTGRES_USER=myuser
POSTGRES_PASSWORD=mypassword
```

And a requirement model which allow you to execute `apt-get install -y postgresql-client`, see [HowTo]({{< relref "workflows/pipelines/requirements/worker-model/docker/_index.md" >}})


![Requirement](/images/tutorials_service_link_pg_requirements.png)

#### Add a step of type `script`

docker image `postgres:9.5.3` start a postgresql at startup. So, it's now available on `mypg`

```bash
#!/bin/bash

set -ex

apt-get update
apt-get install -y postgresql-client

PGPASSWORD=mypassword psql -U myuser -h mypg <<EOF
\x
SELECT version();
EOF
```

![Step](/images/tutorials_service_link_pg_job.png)

**Execute Pipeline**

See output:

![Log](/images/tutorials_service_link_pg_log.png)
