---
title: "Hatchery roles"
weight: 2
---

* `start-worker`: Allow the hatchery to spawn a worker in the given region

Yaml example:
```yaml
name: my-permission-name
hatcheries:
  - role: start-worker
    region: nyc-infra
    hatchery: my-swarm-hatchery
```

List of fields:

* `role`: <b>[mandatory]</b> role to applied
* `region`: <b>[mandatory]</b> the region name
* `hatchery`: <b>[mandatory]</b> the hatchery name
