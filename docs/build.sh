#!/bin/bash

set -e

function generateUserActionsDocumentation {
  for action in `ls ../contrib/actions/*.yml`; do

  echo "work on ${action}"

  filename=$(basename "$action")
  actionName=${filename/.yml/}

  ACTION_FILE="content/docs/actions/${actionName}.md"

  echo "generate ${ACTION_FILE}"

cat << EOF > ${ACTION_FILE}
---
title: "${actionName}"
card:
  name: user-action
---
EOF

  cdsctl action doc ${action} >> $ACTION_FILE

  done;
}

function generatePluginsDocumentation {
  for plugin in `ls ../contrib/grpcplugins/action/*/*.yml`; do

  filename=$(basename "$plugin")
  pluginName=${filename/.yml/}

  OLD=`pwd`
  PLUGIN_FILE="$OLD/content/docs/actions/plugin-${pluginName}.md"

  echo "generate ${PLUGIN_FILE}"

cat << EOF > ${PLUGIN_FILE}
---
title: "plugin-${pluginName}"
card:
  name: plugin
---
EOF

  cdsctl admin plugins doc ${plugin} >> $PLUGIN_FILE

  cd $OLD

  done;
}

function generateBuiltinActionsDocumentation {
  for action in `ls ./actions/builtin-*.part.md`; do

  echo "work on ${action}"

  filename=$(basename "$action")
  actionName=${filename/builtin-/}
  actionName=${actionName/.part.md/}

  ACTION_FILE="content/docs/actions/builtin-${actionName}.md"

  echo "generate ${ACTION_FILE}"

  cdsctl action builtin doc "${actionName}" > $ACTION_FILE

  cat ${action} >> $ACTION_FILE

  done;
}

generateUserActionsDocumentation
generatePluginsDocumentation
generateBuiltinActionsDocumentation
