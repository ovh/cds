## Worker model

Worker models are recipe for worker with a fixed set of capabilities. [Hatcheries](/doc/overview/hatchery.md) start and stop instances of worker models according to API needs.

### Types

There is 2 types of worker models:

 * Docker image (Started by hatchery in mode 'docker', 'swarm', 'mesos')
 * Openstack hosts (Started by hatchery in mode 'openstack')

### Capabilities

Worker models have a fixed set of capabilities associated, allowing CDS engine to pick the best model for each job in the queue.

Matching model capabilities and actions requirements is the root of CDS flexibility.
