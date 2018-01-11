+++
title = "Setup Docker Worker Model with your own image"
weight = 3

+++

A worker model of type `docker` can be spawned by a Hatchery Docker Swarm

## Setup Docker Worker Model with your own image

In this example, we will build a Docker model able to build an AngularJs application with webfonts. To create webfonts, a `grunt` task (optionnally) requires `fontforge` and `ttfautohint`.

The following tools must be included in the model:

* `NodeJs` and `npm`
* `bower`
* `grunt-cli`
* `gulp-cli`
* `fontforge`
* `ttfautohint`

We will use the official **nodejs** image from Docker. In this image, there is already a user named **node**. For the example, we will compile `ttfautohint`.

### Prerequisite

To build a Docker model, you need:

* your favorite text editor
* a sane installation of Docker [https://docs.docker.com/engine/installation/](https://docs.docker.com/engine/installation/)

### Dockerfile

Copy this content in a file named `Dockerfile`

```dockerfile
# User official nodejs docker image
FROM node:6.10.1

#Answer 'yes' to each question
ENV DEBIAN_FRONTEND noninteractive

# Upgrade the debian packages
RUN (apt-get update && apt-get upgrade -y -q && apt-get -y -q autoclean && apt-get -y -q autoremove)

#The official image comes with npm; so we can use it to install some packages
RUN npm install -g grunt-cli gulp-cli bower

# Install fontforge for our specific need
RUN apt-get install -y fontforge

# Install packages and compile ttfautohint (still for our specific need)
RUN apt-get install -y libharfbuzz-dev libfreetype6-dev libqt4-dev\
    && cd /tmp \
    && curl -L http://download.savannah.gnu.org/releases/freetype/ttfautohint-1.6.tar.gz |tar xz\
    && cd ttfautohint-1.6\
    && ./configure\
    && make\
    && make install

# Change user. If you do not specify this command, the user will be root, and in our case,
# bower will shout as it cannot be launched by root
USER node

# Specify a working directory on which the current user has write access
# Remember, a curl command will be, first, executed to download the worker
WORKDIR /home/node
```

### Build it and push it

from you shell, type the following command to build the Docker image:

```
docker build --no-cache --pull -t registry.my.infra.net/my/beautiful/worker:latest .
```

If you want to test it, you can lauch your docker in bash mode :

```
docker run -it registry.my.infra.net/my/beautiful/worker:latest /bin/bash
pwd
fontforge -v
exit
```

Now push it

```
docker push registry.my.infra.net/my/beautiful/worker:latest
```

### Register your model in CDS

* In the UI, click on the wheel on the hand right top corner and select *workers" (or go the the route *#/worker*)
* At the bottom of the page, fill the form
    * Name of your worker *My Beautiful*
    * type *docker*
    * image *registry.my.infra.net/my/beautiful/worker:latest*
* Click on *Add* button and that's it

Now you can specify this model in prerequisite on your job. Create a new prerequisite of type "model", then choose your worker model in list
