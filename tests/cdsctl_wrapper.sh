#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
NANO_TIMESTAMP=$(date +%s%N)

# Prepare args file from current args
rm -f $SCRIPT_DIR/$NANO_TIMESTAMP.args.file
for v in "$@"
do
  echo $v >> $SCRIPT_DIR/$NANO_TIMESTAMP.args.file
done

# Exec cdsctl test binary, capture logs and exit code
$SCRIPT_DIR/cdsctl.test -test.coverprofile=$SCRIPT_DIR/$NANO_TIMESTAMP.coverprofile -test.run="^TestBincoverRunMain$" -args-file=$SCRIPT_DIR/$NANO_TIMESTAMP.args.file 1> $SCRIPT_DIR/$NANO_TIMESTAMP.out 2> $SCRIPT_DIR/$NANO_TIMESTAMP.error.out
EXI=$?

# Print cdsctl log on stdout and go test log in a dedicated file
END_CTL_OUT=false
while IFS="" read -r p || [ -n "$p" ]
do
  if [[ $p == *"START_BINCOVER_METADATA" && "$p" != "START_BINCOVER_METADATA" ]]; then
    P=`printf '%s' "$p" | sed -e "s/START_BINCOVER_METADATA$//"`
    printf '%s\n' "$P"
    END_CTL_OUT=true
    echo "START_BINCOVER_METADATA" >> $SCRIPT_DIR/$NANO_TIMESTAMP.test.out
  else
    if [ "$p" == "START_BINCOVER_METADATA" ]; then
      END_CTL_OUT=true
    fi
    if $END_CTL_OUT; then
      echo "$p" >> $SCRIPT_DIR/$NANO_TIMESTAMP.test.out
    else
      printf '%s\n' "$p"
    fi
  fi
done < $SCRIPT_DIR/$NANO_TIMESTAMP.out

exit $EXI
