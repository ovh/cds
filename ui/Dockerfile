FROM registry.ovh.net/official/node:6.4.0

COPY dist /data
COPY setup /usr/bin/setup

RUN chmod +rx /usr/bin/setup

WORKDIR /data

CMD setup
