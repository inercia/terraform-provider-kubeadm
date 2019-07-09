#!/usr/bin/env bash

SRC_DIRS="pkg internal"

# Check gofmt
echo ">>> Checking that code complies with gofmt requirements..."
gofmt_files=$(gofmt -l `find $SRC_DIRS -name '*.go' | grep -v vendor`)
if [[ -n ${gofmt_files} ]]; then
    echo 'ERROR: gofmt needs running on the following files:'
    echo "${gofmt_files}"
    echo "ERROR: You can use the command: \`make fmt\` to reformat code."
    exit 1
fi

exit 0
