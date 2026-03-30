#!/bin/bash

set -e

# Extra arguments are passed to go test, e.g.:
# ./test-workspace.sh -v

MODULES=$(go list -f '{{.Dir}}' -m | grep -v /tools$)

echo "Found modules:"
echo "$MODULES" | sed 's/^/  /'
echo ""

echo "Building"
echo ""

for dir in $MODULES; do
  (cd "$dir" && go build ./... && go clean ./...)
done

echo ""
echo "Linting with go vet and gosec"
echo ""

for dir in $MODULES; do
  (cd "$dir" && go vet ./...)
done
for dir in $MODULES; do
  case "$dir" in */examples/*) continue ;; esac
  (cd "$dir" && go tool gosec ./...)
done

echo ""
echo "Testing"
echo ""

for dir in $MODULES; do
  (cd "$dir" && go test -p 1 -count=1 "$@" ./...)
done
