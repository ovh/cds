---
title: "Service Requirement NGINX"
weight: 2
card: 
  name: tutorial_requirements
  weight: 1
---

## Add the service requirement

* Name: `mynginx`. This will be the service hostname
* Type: `service`
* Docker Image: `nginx:1.11.1`. This is the name of Docker image to link to current job


![Requirement](/images/tutorials_service_link_nginx_requirements.png)

## Add a step of type `script`

Docker image `nginx:1.11.1` start a nginx at startup. So, it's now available on `http://mynginx`

```bash
curl -v -X GET http://mynginx
```

![Step](/images/tutorials_service_link_nginx_job.png)

**Execute Pipeline**

See output:

![Log](/images/tutorials_service_link_nginx_log.png)
