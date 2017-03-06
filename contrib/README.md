# CDS Contrib

Here you'll find extensions ready to use with [CDS](https://github.com/ovh/cds).

CDS support several kind of extensions:

- Actions
- Plugins
- Templates
- Secret Backends
- µServices

See [CDS documentation](https://github.com/ovh/cds) for more details.

## Actions

- [Docker Package](https://github.com/ovh/cds/tree/master/contrib/actions/cds/cds-docker-package.hcl)
- [Git clone](https://github.com/ovh/cds/tree/master/contrib/actions/cds/cds-git-clone.hcl)
- [Go Build](https://github.com/ovh/cds/tree/master/contrib/actions/cds/cds-go-build.hcl)

## Plugins

- [Kafka Publisher](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-kafka-publish)
- [Mesos/Marathon Deployment](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-marathon)
- [Tmpl](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-tmpl)
- [Mesos/Marathon Group-Tmpl](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-group-tmpl)
- [SSH Cmd](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-ssh-cmd)
- [Venom](https://github.com/ovh/cds/tree/master/contrib/plugins/plugin-venom)

## Templates

- [Plain Template](https://github.com/ovh/cds/tree/master/contrib/templates/cds-template-plain)

## Secret Backends

- [Vault Secret Backend](https://github.com/ovh/cds/tree/master/contrib/secret-backends/secret-backend-vault)

## µServices

- [cds2xmpp](https://github.com/ovh/cds/tree/master/contrib/uservices/cds2xmpp)
- [cds2tat](https://github.com/ovh/cds/tree/master/contrib/uservices/cds2tat)

## Contributions

By convention, plugins must have prefix `plugin`, templates  must have prefix `templates`, secret backends must have prefix `secret-backend`.
