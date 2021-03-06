#!/usr/bin/env bash
set -e

if ! [[ "$0" =~ scripts/awstester.build.release.sh ]]; then
  echo "must be run from repository root"
  exit 255
fi

if [[ -z "${GIT_COMMIT}" ]]; then
  GIT_COMMIT=$(git rev-parse --short=12 HEAD || echo "GitNotFound")
fi

if [[ -z "${RELEASE_VERSION}" ]]; then
  RELEASE_VERSION=$(git rev-parse --short=12 HEAD || echo "GitNotFound")
fi

if [[ -z "${BUILD_TIME}" ]]; then
  BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
fi

echo "GIT_COMMIT:" ${GIT_COMMIT}
echo "RELEASE_VERSION:" ${RELEASE_VERSION}
echo "BUILD_TIME:" ${BUILD_TIME}

mkdir -p ./bin

for os in linux darwin; do
  CGO_ENABLED=0 GOOS=${os} GOARCH=$(go env GOARCH) \
    go build -v \
    -ldflags "-s -w \
    -X github.com/aws/awstester/version.GitCommit=${GIT_COMMIT} \
    -X github.com/aws/awstester/version.ReleaseVersion=${RELEASE_VERSION} \
    -X github.com/aws/awstester/version.BuildTime=${BUILD_TIME}" \
    -o ./bin/awstester-${RELEASE_VERSION}-${os}-$(go env GOARCH) \
    ./cmd/awstester
done

./bin/awstester-${RELEASE_VERSION}-${os}-$(go env GOARCH) version

echo "Success!"
