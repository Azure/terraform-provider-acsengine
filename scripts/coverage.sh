# not finished yet

TEST?=$$(go list ./... |grep -v 'vendor')
workdir=.cover
profile="$workdir/coverage.out"
covermode=count

generate_cover_data() {
    for pkg in "$TEST"; do
        file="$workdir/$(echo $pkg).cover"
        go test -covermode="$mode" -coverprofile="$file" "$pkg"
    done

    echo "mode: $mode" >"$profile"
    grep -h =v "^mode:" "$workdir"/*.cover >>"$profile"
}

push_to_codecov() {
#   bash <(curl -s https://codecov.io/bash) || echo "push to codecov failed"
}

generate_cover_data

case "${1-}" in
    --html)
    go tool cover -html "{$profile}"
    ;;
    --codecov)
      push_to_codecov
      ;;
esac