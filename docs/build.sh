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
  for plugin in `ls ../contrib/grpcplugins/action/plugin-*/*.yml`; do

  filename=$(basename "$plugin")
  pluginName=${filename/.yml/}

  OLD=`pwd`
  PLUGIN_FILE="$OLD/content/docs/actions/${pluginName}.md"

  echo "generate ${PLUGIN_FILE}"

cat << EOF > ${PLUGIN_FILE}
---
title: "${pluginName}"
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

function generateStatic {
  npm ci
  cp node_modules/anchor-js/anchor.min.js static/js/anchor.min.js
  cp node_modules/asciinema-player/dist/bundle/asciinema-player.css static/css/asciinema-player.css
  cp node_modules/asciinema-player/dist/bundle/asciinema-player.min.js static/js/asciinema-player.min.js
  cp node_modules/bootstrap/dist/js/bootstrap.min.js static/js/bootstrap.min.js
  cp node_modules/jquery-ui/dist/jquery-ui.min.js static/js/jquery-ui.min.js
  cp node_modules/jquery/dist/jquery.min.js static/js/jquery.min.js
  cp node_modules/js-autocomplete/auto-complete.min.js static/js/auto-complete.min.js
  cp node_modules/lunr/lunr.min.js static/js/lunr.min.js
}

generateUserActionsDocumentation
generatePluginsDocumentation
generateBuiltinActionsDocumentation
generateStatic
