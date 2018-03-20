#!/bin/bash

set -e

function generateUserActionsDocumentation {
  for action in `ls ../contrib/actions/*.hcl`; do

  filename=$(basename "$action")
  actionName=${filename/.hcl/}

  mkdir -p content/workflows/pipelines/actions/user
  ACTION_FILE="content/workflows/pipelines/actions/user/${actionName}.md"

  echo "generate ${ACTION_FILE}"

cat << EOF > ${ACTION_FILE}
+++
title = "${actionName}"

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

  mkdir -p content/workflows/pipelines/actions/plugins
  OLD=`pwd`
  PLUGIN_FILE="$OLD/content/workflows/pipelines/actions/plugins/${plugin}.md"

  cd ../contrib/plugins/${plugin}

  echo "Compile plugin ${plugin}"
  go build

  echo "generate ${PLUGIN_FILE}"

cat << EOF > ${PLUGIN_FILE}
+++
title = "${plugin}"

+++
EOF

  ./${plugin} info >> $PLUGIN_FILE

  cd $OLD

  done;
}

generateUserActionsDocumentation
generatePluginsDocumentation
