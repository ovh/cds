---
title: "Secret Requirement"
weight: 4
---

The `Secret` prerequisite allows you to require a worker to start with some project's secrets when those secrets are not automatically injected.

Secret automatic injection can be disabled if a job requires to run in a specific region (using a "Region" prerequisite) that was added in CDS API configuration (key: skipProjectSecretsOnRegion).

The value for the requirement should be a valid regex. In the following example it is used to match both default SSH and PGP keys for a CDS project.

Example of job configuration:
```
- job: build
  requirements:
  - region: myregion
  - secret: ^cds.key.proj-(ssh|pgp)-test.priv$
  steps:
  ...
```

Example of CDS API configuration:
```
[api]
  ...
  [api.secrets]
    ...
    skipProjectSecretsOnRegion = ["myregion"]
```

Example of CDS Hatchery configuration:
```
[hatchery]
  [hatchery.local]
    ...
    [hatchery.local.commonConfiguration]
      ...
      [hatchery.local.commonConfiguration.provision]
        ...
        region = "myregion"
        ignoreJobWithNoRegion = true
```
