This directory contains some examples of worker models.

# What's a worker model?

See https://ovh.github.io/cds/workflows/pipelines/requirements/worker-model/

# How to import a worker model?

```bash
# import a worker model
cdsctl worker model import ./go-official-1.11.4-stretch.yml

# or with a remote file
cdsctl worker model import https://raw.githubusercontent.com/ovh/cds/master/contrib/worker-models/go-official-1.11.4-stretch.yml

```

