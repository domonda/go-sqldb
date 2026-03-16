#!/bin/bash

# Just in case the script is run from another directory
SCRIPT_DIR=$(cd -P -- $(dirname -- "$0") && pwd -P)
cd $SCRIPT_DIR

MODULE_PATHS=("" "mssqlconn/" "mysqlconn/" "pqconn/" "sqliteconn/")

# Show current tags and usage if no arguments provided
if [ -z "$1" ]; then
    echo "Current release tags:"
    echo ""

    for PREFIX in "${MODULE_PATHS[@]}"; do
        if [ -z "$PREFIX" ]; then
            MODULE_NAME="(root module)"
        else
            MODULE_NAME="${PREFIX%/}"
        fi
        echo "  Module: $MODULE_NAME"

        # Get the latest tag for this module
        LATEST_TAG=$(git tag -l "${PREFIX}v*" --sort=-v:refname | head -1)
        if [ -n "$LATEST_TAG" ]; then
            echo "    Latest: $LATEST_TAG"
            # Show last 3 tags for this module
            echo "    Recent:"
            git tag -l "${PREFIX}v*" --sort=-v:refname | head -3 | sed 's/^/      /'
        else
            echo "    No tags yet"
        fi
        echo ""
    done

    echo "Usage: $0 <version> [message]"
    echo ""
    echo "Creates tags for all modules with the specified version."
    echo ""
    echo "Examples:"
    echo "  $0 v0.99.1               # Creates v0.99.1, cmd/sqldb-dump/v0.99.1, mssqlconn/v0.99.1, mysqlconn/v0.99.1, pqconn/v0.99.1, sqliteconn/v0.99.1"
    echo "  $0 v0.99.1 \"bug fixes\"   # Same with custom message"
    echo "  $0 v1.0.0-beta1          # Pre-release version"
    echo ""
    echo "Version format: vMAJOR.MINOR.PATCH[-PRERELEASE]"
    exit 0
fi

SEMVER_REGEX='^v([0-9]+)\.([0-9]+)\.([0-9]+)(-[a-zA-Z0-9]+)?$'
if [[ ! "$1" =~ $SEMVER_REGEX ]]; then
    echo "Error: Invalid version format: $1"
    echo ""
    echo "Usage: $0 <version> [message]"
    echo "Version must be in format: vMAJOR.MINOR.PATCH[-PRERELEASE]"
    echo "Examples: v0.99.1, v1.0.0, v2.1.3-beta1"
    exit 1
fi

VERSION=$1
MESSAGE="$VERSION"
if [ -n "$2" ]; then
    MESSAGE="$2" # provided as the second argument
fi

echo "Tagging $VERSION with message: '$MESSAGE'"

for PREFIX in "${MODULE_PATHS[@]}"; do
    echo "  tag ${PREFIX}${VERSION}"
    git tag -a "${PREFIX}${VERSION}" -m "$MESSAGE"
done

echo "Tags to be pushed"
git push --tags --dry-run

echo "Do you want to push tags to origin? (y/n)"
read CONFIRM
if [[ "$CONFIRM" == "y" || "$CONFIRM" == "Y" ]]; then
    git push origin --tags

    # Fetch modules via Go proxy in background to update pkg.go.dev cache
    MODULE_BASE="github.com/domonda/go-sqldb"
    (
        for PREFIX in "${MODULE_PATHS[@]}"; do
            if [ -z "$PREFIX" ]; then
                MODULE="$MODULE_BASE"
            else
                MODULE="$MODULE_BASE/${PREFIX%/}"
            fi
            GOPROXY=proxy.golang.org go list -m "$MODULE@$VERSION" 2>/dev/null
        done
    ) &>/dev/null &
else
    for PREFIX in "${MODULE_PATHS[@]}"; do
        git tag -d "${PREFIX}${VERSION}"
    done
    echo "Reverted local $VERSION tags"
fi