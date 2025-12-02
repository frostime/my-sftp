#!/bin/bash
set -euo pipefail  # 错误时退出，未定义变量报错，管道任一失败则失败

VERSION=$(git describe --tags --always)
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "Building my-sftp version ${VERSION} (commit ${COMMIT}) on ${DATE}"

go build -ldflags "-s -w -X 'main.Version=${VERSION}' -X 'main.Commit=${COMMIT}' -X 'main.Date=${DATE}'" -o my-sftp

echo "✓ Build successful: my-sftp"
