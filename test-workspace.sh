#!/bin/bash

set -e

echo "Linting"
echo ""

go list -f '{{.Dir}}' -m | xargs -I {} go vet {}/...
go list -f '{{.Dir}}' -m | grep -v /cmd/ | xargs -I {} sh -c 'output=$(go tool gosec {}/... 2>/dev/null) || { printf "%s\n" "$output"; exit 1; }'

echo ""
echo "Testing"
echo ""

go list -f '{{.Dir}}' -m | xargs -I {} go test {}/...
