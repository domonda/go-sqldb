#!/bin/bash

set -e

# Extra arguments are passed to go test, e.g.:
# ./test-workspace.sh -v

echo "Found modules:"
go list -f '  {{.Dir}}' -m | grep -v /tools$
echo ""

echo "Building"
echo ""

go list -f '{{.Dir}}' -m | grep -v /tools$ | xargs -I {} go build {}/...
go list -f '{{.Dir}}' -m | grep -v /tools$ | xargs -I {} go clean {}/...

echo ""
echo "Linting with go vet and gosec"
echo ""

go list -f '{{.Dir}}' -m | grep -v /tools$ | xargs -I {} go vet {}/...
go list -f '{{.Dir}}' -m | grep -v /tools$ | grep -v /examples/ | xargs -I {} go tool gosec {}/...

echo ""
echo "Testing"
echo ""

go list -f '{{.Dir}}' -m | grep -v /tools$ | xargs -I {} go test -p 1 -count=1 "$@" {}/...
