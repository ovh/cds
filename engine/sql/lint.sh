#!/bin/bash
set -e

## count if there is many SQL files with the same number
ERROR=$(ls $1/*.sql | cut -d '_' -f1 | uniq -c | grep -v '1 ' | awk 'END {print NR}')
if [[ "x$ERROR" != "x0" ]]; then
	echo "Please check the prefix number on '$1' sql files, seems two files have to same prefix."
	exit 1
fi;
