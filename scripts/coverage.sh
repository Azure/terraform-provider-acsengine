#!/usr/bin/env bash

set -eo pipefail

TEST="./acsengine"
# TEST="./acsengine ./acsengine/utils ./acsengine/helpers/client" # should have utils and everything in helpers too...
# TEST=$(go list ./... |grep -v 'vendor')
# TEST=$(find ./acsengine -type d)
coverdir=$(mktemp -d /tmp/coverage.XXXXXXXXXX)
profile="coverage.out"
mode="count"

generate_cover_data() {
    for pkg in $TEST; do
        echo $pkg
        file="$coverdir/$pkg.cover"
        go test "$pkg" -covermode="$mode" -coverprofile="$file"
    done
    # find ./acsengine -type f -name "*.go" | while read -r file; do echo $file; go test $file -covermode="$mode" -coverprofile="$file".cover && mv "$file".cover ${coverdir}; done
    # find ./acsengine -type f -name "*.go" | while read -r file; do echo $file; done

    echo "mode: $mode" >"$profile" # somehow this read into scale_k8s_cluster.go when I used second option
    grep -h -v "^mode:" "$coverdir"/*.cover >>"$profile"
}

push_to_codecov() {
  bash <(curl -s https://codecov.io/bash) || echo "push to codecov failed"
}

generate_cover_data
go tool cover -func "$profile"

case "${1-}" in
    --html)
    go tool cover -html "{$profile}"
    ;;
    --codecov)
    push_to_codecov
    ;;
esac