#!/bin/bash

set -e

function generateUserActionsDocumentation {
  for action in `ls ../contrib/actions/*.hcl`; do

  filename=$(basename "$action")
  actionName=${filename/.hcl/}

  ACTION_FILE="content/building-pipelines/building-pipelines.actions.user.${actionName}.md"

  echo "generate ${ACTION_FILE}"

cat << EOF > ${ACTION_FILE}
+++
title = "${actionName}"

[menu.main]
parent = "actions-user"
identifier = "${actionName}"

+++
EOF

  cds -w action doc ${action} >> $ACTION_FILE

  done;
}

function generatePluginsDocumentation {
  for plugin in `ls ../contrib/plugins/`; do

  if [[ "${plugin}" != plugin-* ]]; then
    echo "skip ../contrib/plugins/${plugin}"
    continue;
  fi

  OLD=`pwd`
  PLUGIN_FILE="$OLD/content/building-pipelines/building-pipelines.actions.plugins.${plugin}.md"

  cd ../contrib/plugins/${plugin}

  echo "Compile plugin ${plugin}"
  go build

  echo "generate ${PLUGIN_FILE}"

cat << EOF > ${PLUGIN_FILE}
+++
title = "${plugin}"

[menu.main]
parent = "actions-plugins"
identifier = "${plugin}"

+++
EOF

  ./${plugin} info >> $PLUGIN_FILE

  cd $OLD

  done;
}

generateUserActionsDocumentation
generatePluginsDocumentation
