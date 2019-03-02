#!/bin/bash

set -e

function generateUserActionsDocumentation {
  for action in `ls ../contrib/actions/*.yml`; do

  echo "work on ${action}"

  filename=$(basename "$action")
  actionName=${filename/.yml/}

  ACTION_FILE="content/manual/actions/${actionName}.md"

  echo "generate ${ACTION_FILE}"

cat << EOF > ${ACTION_FILE}
+++
title = "${actionName}"

+++
EOF

  cdsctl action doc ${action} >> $ACTION_FILE

  done;
}

function generatePluginsDocumentation {
  for plugin in `ls ../contrib/grpcplugins/action/*/*.yml`; do

  filename=$(basename "$plugin")
  pluginName=${filename/.yml/}

  OLD=`pwd`
  PLUGIN_FILE="$OLD/content/manual/actions/plugin-${pluginName}.md"

  echo "generate ${PLUGIN_FILE}"

cat << EOF > ${PLUGIN_FILE}
+++
title = "plugin-${pluginName}"

+++
EOF

  cdsctl admin plugins doc ${plugin} >> $PLUGIN_FILE

  cd $OLD

  done;
}

generateUserActionsDocumentation
generatePluginsDocumentation
