FROM node:10.13.0-jessie

# Dockerfile inspired from https://github.com/RocketChat/hubot-rocketchat/blob/master/Dockerfile

RUN npm install -g coffee-script yo generator-hubot && useradd hubot -m

USER hubot

WORKDIR /home/hubot

ENV BOT_NAME "cdsbot"
ENV BOT_OWNER "CDS Team"
ENV BOT_DESC "Hubot with xmpp adapter"

ENV EXTERNAL_SCRIPTS=hubot-diagnostics,hubot-help,hubot-rules,hubot-shipit

RUN yo hubot --owner="$BOT_OWNER" --name="$BOT_NAME" --adapter="xmpp" --description="$BOT_DESC" --defaults && \
	node -e "console.log(JSON.stringify('$EXTERNAL_SCRIPTS'.split(',')))" > external-scripts.json && \
	npm install hubot-scripts

# hack added to get around owner issue: https://github.com/docker/docker/issues/6119
USER root
ADD scripts/*.coffee /home/hubot/scripts/
RUN chown hubot:hubot -R /home/hubot/
USER hubot

CMD bin/hubot -n $BOT_NAME -a xmpp
