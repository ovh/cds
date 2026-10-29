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
* a MicroVM with the kernel and rootfs of the worker model, the vCPUs, memory
  and disk of its [flavor](#flavors), the egress
  policy of the hatchery configuration, and `spec.guest` asking the platform
  to write the Secret keys as guest files and to enable the unit. When the
  hatchery could compute its sha256, the worker binary of the model os/arch
  is a guest file too, with the API download route as source: the platform
  downloads it once per node and per CDS version. The binaries of the
  `preloadedPlugins` (default `script`, which runs the `run` steps) are guest
  files the same way, written in the worker basedir under their published
  name: the worker checks their sha512 and does not download them at the
  first step. A plugin the hatchery cannot check against its published
  sha512 is left to the worker.

The worker basedir is `/var/lib/cds-worker` unless `provision.workerBasedir`
is set: `/tmp` is emptied at boot.

The unit starts as soon as the platform network unit (`microvm-net.service`)
is done, without waiting for the rest of the guest boot. If the worker binary
is missing, it downloads it from the CDS API (`/download/worker/<os>/<arch>`),
then runs it. When the worker exits, the guest
reboots, which ends the microVM. The Secret is owned by the MicroVM and is
deleted with it.

The kernel command line is the platform default followed by `extraBootArgs`,
which defaults to `quiet`, kernel arguments that skip what a microVM does not
need at boot (`i8042.*` for the absent PS/2 controller, `tsc=reliable`,
`no_timer_check`, `random.trust_cpu=on`), and `systemd.mask=` of units a CI
worker does not need (getty, apt and maintenance timers). The SSH server is masked too unless
`debugSSHAuthorizedKeys` is set.

The guest image needs systemd, bash and curl. Its egress policy must allow the
CDS API and the CDN (HTTP and TCP log port).

Workers have no SSH access. As a debugging aid, setting
`debugSSHAuthorizedKeys` in the hatchery configuration enables the admin
network of every new worker and authorizes these keys for root (SSH from the
bastion). Remove it once done.

## Flavors

A flavor is a worker size: vCPUs, memory and disk. The hatchery
configuration declares them; without any, it uses:

| Flavor | vCPUs | Memory (MiB) | Disk (GiB) |
|--------|-------|--------------|------------|
| XS     | 1     | 4096         | 10         |
| S      | 2     | 8192         | 20         |
| M      | 4     | 16384        | 50         |
| L      | 6     | 24576        | 100        |

A worker gets the first flavor set among:

1. the job flavor, `runs-on.flavor`;
2. the worker model flavor, `spec.flavor`;
3. the hatchery `defaultFlavor`, `XS` by default.

Names are case-insensitive. The hatchery does not take a job whose flavor it
does not know, and logs a warning naming the flavor and the known ones: another
hatchery may know it. The job page shows the flavor the worker got.

The VM disk is the flavor disk, raised to `spec.rootfs.size` of the model when
that is larger: the platform does not convert an image to a smaller disk.

```yaml
jobs:
  build:
    runs-on:
      model: my-microvm-model
      flavor: M
```

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
  # Worker sizes, the four default flavors when none is declared
  defaultFlavor = "S"
  # Out of the cluster only; the in-cluster configuration is used when empty
  # kubernetesConfigFile = "/etc/cds/kubeconfig"

  [[hatchery.microvm.flavors]]
    name = "S"
    vcpus = 2
    memoryMiB = 8192
    diskGiB = 20

  [[hatchery.microvm.flavors]]
    name = "L"
    vcpus = 6
    memoryMiB = 24576
    diskGiB = 100

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
