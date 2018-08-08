#!/usr/bin/env bash

set -eo pipefail

test=$(go list ./... |grep -v 'vendor')
coverdir=$(mktemp -d /tmp/coverage.XXXXXXXXXX)
profile="coverage.out"
mode="count"

generate_cover_data() {
    count=1
    for pkg in $test; do
        count=$count+1
        file="$coverdir/$count.cover"
        go test "$pkg" -covermode="$mode" -coverprofile="$file"
    done

    echo "mode: $mode" >"$profile"
    grep -h -v "^mode:" "$coverdir"/*.cover >>"$profile"
}

push_to_codecov() {
  bash <(curl -s https://codecov.io/bash) || echo "push to codecov failed"
}

generate_cover_data
go tool cover -func=$profile

case "${1-}" in
    --html)
    go tool cover -html "{$profile}"
    ;;
    --codecov)
    push_to_codecov
    ;;
esac