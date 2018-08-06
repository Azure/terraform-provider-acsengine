#!/usr/bin/env bash

set -eo pipefail

# TEST=$(go list ./... |grep -v 'vendor')
TEST="./acsengine"
coverdir=$(mktemp -d /tmp/coverage.XXXXXXXXXX)
profile="$coverdir/coverage.out"
mode="count"

generate_cover_data() {
    for pkg in "$TEST"; do
        file="$coverdir/$(echo $pkg).cover"
        go test -covermode="$mode" -coverprofile="$file" "$pkg"
    done

    echo "mode: $mode" >"$profile"
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