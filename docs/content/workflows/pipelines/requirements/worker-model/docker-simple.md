+++
title = "Setup Worker Model From Docker Hub"
weight = 2

+++

A worker model of type `docker` can be spawned by a Hatchery Docker Swarm.

## Register a worker Model from an existing Docker Image

Docker Image *golang:1.8.1* have a "curl" in $PATH, so it can be used as it is.

* In the UI, click on the wheel on the hand right top corner and select *workers" (or go the the route *#/worker*)
* At the bottom of the page, fill the form
    * Name of your worker *Golang-1.8.1*
    * type *docker*
    * image *golang:1.8.1*
* Click on *Add* button and that's it
