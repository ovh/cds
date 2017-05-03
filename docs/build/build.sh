#!/bin/bash

set -e

function generateUserActionDocumentation {
  for action in `ls ../../contrib/actions/*.hcl`; do

  filename=$(basename "$action")
  actionName=${filename/.hcl/}

  ACTION_FILE="content/building-pipelines/actions/user/${actionName}.md"

  echo "generate ${ACTION_FILE}"

cat << EOF > ${ACTION_FILE}
+++
title = "${actionName}"
chapter = true

[menu.main]
parent = "actions-user"
identifier = "${actionName}"

+++
EOF

  cds action doc ../../contrib/actions/${action} >> $ACTION_FILE

  done;
}

function generatePluginDocumentation {
  for plugin in `ls ../../contrib/plugins/`; do

  if [[ "${plugin}" != plugin-* ]]; then
    echo "skip ../../contrib/plugins/${plugin}"
    continue;
  fi

  OLD=`pwd`
  PLUGIN_FILE="$OLD/content/building-pipelines/actions/plugins/${plugin}.md"

  cd ../../contrib/plugins/${plugin}

  echo "Compile plugin ${plugin}"
  go build

  echo "generate ${PLUGIN_FILE}"

cat << EOF > ${PLUGIN_FILE}
+++
title = "${plugin}"
chapter = true

[menu.main]
parent = "actions-plugins"
identifier = "${plugin}"

+++
EOF

  ./${plugin} info >> $PLUGIN_FILE

  cd $OLD

  done;
}

generateUserActionDocumentation

generatePluginDocumentation

hugo -d ../
hugo server
