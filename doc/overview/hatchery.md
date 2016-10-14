# Hatchery

Hatchery is a binary dedicated to spawn and kill worker in accordance with build queue needs.

There is 5 modes for hatcheries:

 * Local (Start workers on a single host)
 * Local Docker (Start worker model instances on a single host)
 * Mesos (Start worker model instances on a mesos cluster)
 * Swarm (Start worker on a docker swarm cluster)
 * Openstack (Start hosts on an openstack cluster)

## Local mode

Hatchery starts workers directly as local process.

## Docker mode

Hatchery starts workers inside docker containers on the same host. Setup tutorial [here](/doc/tutorials/first-hatchery.md)

## Marathon mode

Hatchery starts workers inside containers on a mesos cluster using Marathon API.

## Openstack mode

Hatchery starts workers on Openstack servers using Openstack Nova.

## Swarm mode

The hatchery connects to a swarm cluster and starts workers inside containers. 
