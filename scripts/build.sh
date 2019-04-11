#!/bin/bash
# mass: build shell script

## Usage:
##       ./build.sh {version} {target}
##           version: string. required. the version for the built binary filename.
##           target: string. optional. if target is "CURRENT", the script will build ONLY for current OS/ARCH.

PACKAGE_NAME="mass-wallet"
# 1. get version flag
if [ -z "$1" ]
  then
    echo "No version provided."
    exit 1
fi
echo "Building version $1..."

# 2. build for current platform
if [ "$2" = "CURRENT" ]
    then
      BINARY_OUTPUT_NAME='./bin/'$PACKAGE_NAME-$1
      if [ "$3" = "DEBUG" ]
        then
        go build -gcflags "-N -l" -ldflags "-X massnet.org/mass-wallet/version.GitCommit=`git rev-parse HEAD`" -o $BINARY_OUTPUT_NAME
      else
        go build -gcflags "-N -l" -ldflags "-X massnet.org/mass-wallet/version.GitCommit=`git rev-parse HEAD`" -o $BINARY_OUTPUT_NAME
      fi
      
      echo "    $BINARY_OUTPUT_NAME"
    exit
fi

# 3. build for all platforms

PLATFORMS=("windows/amd64" "darwin/amd64" "linux/amd64" "linux/arm")

for PLATFORM in "${PLATFORMS[@]}"
do
    PLATFORM_SPLIT=(${PLATFORM//\// })
    GOOS=${PLATFORM_SPLIT[0]}
    GOARCH=${PLATFORM_SPLIT[1]}
    BINARY_OUTPUT_NAME='./bin/'$PACKAGE_NAME-$1-$GOOS-$GOARCH
    
    if [ $GOOS = "windows" ]; then
        BINARY_OUTPUT_NAME+='.exe'
    fi

    echo "Building for $GOOS/$GOARCH ..."
    env GOOS=$GOOS GOARCH=$GOARCH go build -gcflags "-N -l" -ldflags "-X massnet.org/mass-wallet/version.GitCommit=`git rev-parse HEAD`" -o $BINARY_OUTPUT_NAME
    echo "    $BINARY_OUTPUT_NAME"

done
