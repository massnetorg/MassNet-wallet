#!/bin/bash
# mass: release shell script

## Usage:
##       ./release.sh {version} {target}
##           version: string. required. the version for the built binary filename.

RELEASE_NAME="mass"

# 1. get version flag
if [ -z "$1" ]
  then
    echo "No version provided."
    exit 1
fi
echo "Building version $1..."

# 2. build for all platforms

PLATFORMS=("windows/amd64" "darwin/amd64" "linux/amd64" "linux/arm")

for PLATFORM in "${PLATFORMS[@]}"
do
    PLATFORM_SPLIT=(${PLATFORM//\// })
    GOOS=${PLATFORM_SPLIT[0]}
    GOARCH=${PLATFORM_SPLIT[1]}

    BINARY_OUTPUT_EXT=''
    if [ $GOOS = "windows" ]; then
        BINARY_OUTPUT_EXT+='.exe'
    fi
    BINARY_OUTPUT_DIR='./bin/release/'$GOOS-$GOARCH
    BINARY_OUTPUT_NAME_MAIN=$BINARY_OUTPUT_DIR'/'$RELEASE_NAME'-wallet'$BINARY_OUTPUT_EXT
    BINARY_OUTPUT_NAME_CMD=$BINARY_OUTPUT_DIR'/'$RELEASE_NAME'cli'$BINARY_OUTPUT_EXT

    echo "Building for $GOOS/$GOARCH ..."
    env GOROOT_FINAL=/dev/null GOOS=$GOOS GOARCH=$GOARCH go build -gcflags=-trimpath=$GOPATH/src/massnet.org/mass-wallet -asmflags=-trimpath=$GOPATH/src/massnet.org/mass-wallet -ldflags "-s -w -X massnet.org/mass-wallet/version.GitCommit=`git rev-parse HEAD` -X massnet.org/mass-wallet/consensus.UserTestNetStr=true" -o $BINARY_OUTPUT_NAME_MAIN
    echo "    $BINARY_OUTPUT_NAME_MAIN"
    env GOROOT_FINAL=/dev/null GOOS=$GOOS GOARCH=$GOARCH go build -gcflags=-trimpath=$GOPATH/src/massnet.org/mass-wallet -asmflags=-trimpath=$GOPATH/src/massnet.org/mass-wallet -ldflags "-s -w" -o $BINARY_OUTPUT_NAME_CMD cmd/masscli/main.go
    echo "    $BINARY_OUTPUT_NAME_CMD"
    cp sample-config.json $BINARY_OUTPUT_DIR/config.json

done
