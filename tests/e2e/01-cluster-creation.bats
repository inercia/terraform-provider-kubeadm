source config.bash
[ -f dynamic.bash ] && source dynamic.bash
[ -f local.bash   ] && source local.bash

load helpers

@test "cluster creation" {
  run echo "hello world"
  [ "$status" -eq 0 ]
  [ "$output" = "hello world" ]
}

