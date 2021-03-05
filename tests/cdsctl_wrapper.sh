#!/bin/bash

NANO_TIMESTAMP=$(date +%s%N)

# Prepare args file from current args
rm -f ./$NANO_TIMESTAMP.args.file
for v in "$@"
do
  echo $v >> ./$NANO_TIMESTAMP.args.file
done

# Exec cdsctl test binary, capture logs and exit code
./cdsctl.test -test.coverprofile=./$NANO_TIMESTAMP.coverprofile -test.run="^TestBincoverRunMain$" -args-file=./$NANO_TIMESTAMP.args.file 1> $NANO_TIMESTAMP.out 2> $NANO_TIMESTAMP.error.out
EXI=$?

# Print cdsctl log on stdout and go test log in a dedicated file
END_CTL_OUT=false
while IFS="" read -r p || [ -n "$p" ]
do
  if [[ $p == *"START_BINCOVER_METADATA" && "$p" != "START_BINCOVER_METADATA" ]]; then
    P=`printf '%s' "$p" | sed -e "s/START_BINCOVER_METADATA$//"`
    printf '%s\n' "$P"
    END_CTL_OUT=true
    echo "START_BINCOVER_METADATA" >> $NANO_TIMESTAMP.test.out
  else
    if [ "$p" == "START_BINCOVER_METADATA" ]; then
      END_CTL_OUT=true
    fi
    if $END_CTL_OUT; then
      echo "$p" >> $NANO_TIMESTAMP.test.out
    else
      printf '%s\n' "$p"
    fi
  fi
done < ./$NANO_TIMESTAMP.out

exit $EXI
