+++
title = "Update your worker"
weight = 2

+++

### Manual Update

If you run manually a worker, you probably want to update it when CDS Engine is updated with a new Release.

Update your worker from CDS API:

```bash
./worker update --api https://your.cds.instance
```

Update your worker from latest Release from GitHub:

```bash
./worker update --from-github
```

### Auto Update

If you use a dedicated worker, you launch it with the command:

```bash
./worker --api https://your.cds.instance
```

You can add `auto-update` flag, to auto update the worker, without restart it.

```bash
./worker --api https://your.cds.instance --auto-update
```

