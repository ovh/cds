---
title: "Hatchery MicroVM"
weight: 2
---

## Use case

The MicroVM hatchery runs each CDS v2 job in its own Firecracker microVM. It
does not talk to Firecracker: it creates a `MicroVM` custom resource
(`microvm.poc/v1alpha1`) in a Kubernetes namespace, and the platform behind
this resource schedules, boots and stops the VM. Kubernetes quotas of the
namespace bound the number and the size of the workers.

The hatchery only handles v2 jobs whose worker model has the type `microvm`
(see [worker model]({{< relref "/docs/concepts/cds_as_code/entities/worker_model.md" >}})).

## How a worker starts

For each job, the hatchery creates:

* a Secret holding the worker configuration, a boot script and a systemd unit,
  labelled `microvm.poc/guest-files: "true"` as the platform requires;
* a MicroVM with the kernel, rootfs and size of the worker model, the egress
  policy of the hatchery configuration, and `spec.guest` asking the platform
  to write the Secret keys as guest files and to enable the unit. When the
  hatchery could compute its sha256, the worker binary of the model os/arch
  is a guest file too, with the API download route as source: the platform
  downloads it once per node and per CDS version.

The unit starts as soon as the platform network unit (`microvm-net.service`)
is done, without waiting for the rest of the guest boot. If the worker binary
is missing, it downloads it from the CDS API (`/download/worker/<os>/<arch>`),
then runs it. When the worker exits, the guest
reboots, which ends the microVM. The Secret is owned by the MicroVM and is
deleted with it.

The kernel command line is the platform default followed by `extraBootArgs`,
which defaults to `quiet` and `systemd.mask=` of units a CI worker does not
need (getty, apt and maintenance timers). The SSH server is masked too unless
`debugSSHAuthorizedKeys` is set.

The guest image needs systemd, bash and curl. Its egress policy must allow the
CDS API and the CDN (HTTP and TCP log port).

Workers have no SSH access. As a debugging aid, setting
`debugSSHAuthorizedKeys` in the hatchery configuration enables the admin
network of every new worker and authorizes these keys for root (SSH from the
bastion). Remove it once done.

## Lifecycle

The hatchery lists its MicroVMs by label and deletes:

* the ended ones (phase `Succeeded` or `Failed`), logging their reason;
* the ones without a registered worker `registrationTimeout` seconds after the
  boot;
* the ones refused by the namespace quota (`Pending` with `QuotaExceeded`)
  for more than `quotaPendingTimeout` seconds.

While one of its MicroVMs is refused by the quota, the hatchery does not
create new ones. `/mon/status` shows the number of MicroVMs waiting for the
quota.

## Start the hatchery

The hatchery is v2 only: create a hatchery consumer and set its token in
`hatchery.microvm.commonConfiguration.api.tokenV2`.

```bash
cdsctl experimental hatchery add my-microvm-hatchery
```

Generate the configuration with `engine config new hatchery:microvm`, then
set at least:

```toml
[hatchery.microvm]
  # Namespace of the MicroVMs, their Secrets and the egress policy
  namespace = "cds-workers"
  # MicroVMEgressPolicy applied to every worker
  egressPolicy = "cds-workers"
  # Size used when the model does not set it, and maximum size accepted
  defaultVCPUs = 2
  defaultMemory = 2048
  maxVCPUs = 8
  maxMemory = 16384
  # Out of the cluster only; the in-cluster configuration is used when empty
  # kubernetesConfigFile = "/etc/cds/kubeconfig"

  [hatchery.microvm.commonConfiguration]
    name = "my-microvm-hatchery"
    osArch = ["linux/amd64"]

    [hatchery.microvm.commonConfiguration.api]
      tokenV2 = "xxx"

    [hatchery.microvm.commonConfiguration.provision]
      maxWorker = 10
```

```bash
engine start hatchery:microvm --config config.toml
```

The Kubernetes identity of the hatchery needs `create`, `get`, `list` and
`delete` on `microvms.microvm.poc`, and `create`, `patch`, `list` and `delete`
on `secrets`, in the namespace.
