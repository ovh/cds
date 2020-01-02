# Tests plan

## Installation tests

+ Download latest release from github (optionnal)
+ Assert binaries: engine, cdsctl, worker
+ Assert third parties: postgres, redis
+ Run smtpmock to mock smtp server
+ Generate new config with `engine config new`
+ Modify config with `engine config set`
+ Setup database
