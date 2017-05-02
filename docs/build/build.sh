#!/bin/bash


set -e

for plugin in `ls ../../contrib/plugins/`; do

# it's not a directory
if [ ! -d "../../contrib/plugins/plugin/${plugin}" ]; then
  continue;
fi

PLUGIN_FILE="content/building-pipelines/actions/plugins/${plugin}.md"

if [ -f ${PLUGIN_FILE} ]; then
  continue;
else
  echo "file ${PLUGIN_FILE} already exists"
fi;

echo "generate ${PLUGIN_FILE}"

cat << EOF > ${PLUGIN_FILE}
+++
title = "${plugin}"
chapter = true

[menu.main]
parent = "actions-plugins"
identifier = "${plugin}"

+++

### More

More documentation on [Github](https://github.com/ovh/cds/tree/master/contrib/plugins/${plugin}.md)

EOF

done;

hugo -d ../ ; hugo server
