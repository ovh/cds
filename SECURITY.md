
 # Security Policy

 ## Supported versions

 Security fixes are made on the `master` branch and shipped in the next
 release. Released versions do not receive backported fixes: upgrade to the
 [latest release](https://github.com/ovh/cds/releases/latest) to get them.

 ## Reporting a vulnerability

 Please do not report security vulnerabilities through public GitHub issues,
 discussions or pull requests.

 Report them privately through GitHub:
 [Report a vulnerability](https://github.com/ovh/cds/security/advisories/new).

 Please include:

 - the CDS version (`cdsctl version`, or the version shown in the UI)
 - the affected component: API, UI, CDN, hooks, hatchery, worker, cdsctl,
   or a contrib integration
 - how CDS is deployed (binaries, `ovhcom/cds-engine` Docker image, ...)
 - steps to reproduce, and the impact you observed or expect

 ## What to expect

 We will acknowledge your report as soon as possible and keep you informed
 of the progress towards a fix. Once the fix is released, we publish a GitHub
 security advisory and credit you, unless you prefer to stay anonymous.

 ## Scope

 In scope: the code of this repository and the artifacts published from it
 (GitHub release binaries and the `ovhcom/cds-engine` Docker image).

 Out of scope: the configuration of a self-hosted CDS instance, the
 `docker-compose.yml` of this repository (a local demo, not a production
 setup), and vulnerabilities in third party services CDS integrates with.

 Facts checked in the clone: no SECURITY.md anywhere, .github/ holds
 CODEOWNERS, copilot-instructions.md and PULL_REQUEST_TEMPLATE.md, published
 image ovhcom/cds-engine (README, docs), README sends users to GitHub
 Discussions, components under engine/ (api, ui, cdn, hooks, hatchery,
 worker, ...) and cli/ (cdsctl).

 To confirm while writing: the exact command printing the version
 (cdsctl version assumed), from cli/cdsctl.
