#!/usr/bin/env bash
set -euo pipefail

SKIP_UI=0
SKIP_GO=0
GOARCH_VALUE="${GOARCH_VALUE:-amd64}"
OUTPUT_DIR="${OUTPUT_DIR:-bin/windows-${GOARCH_VALUE}}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-ui)
      SKIP_UI=1
      shift
      ;;
    --skip-go)
      SKIP_GO=1
      shift
      ;;
    --arch)
      GOARCH_VALUE="${2:?missing value for --arch}"
      OUTPUT_DIR="bin/windows-${GOARCH_VALUE}"
      shift 2
      ;;
    --output-dir)
      OUTPUT_DIR="${2:?missing value for --output-dir}"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      echo "Usage: $0 [--skip-ui] [--skip-go] [--arch amd64|386|arm64] [--output-dir path]" >&2
      exit 1
      ;;
  esac
done

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "Required command not found: $1" >&2
    exit 1
  fi
}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
UI_DIR="${SCRIPT_DIR}/ui"
BIN_DIR="${SCRIPT_DIR}/${OUTPUT_DIR}"

echo "Repo root: ${SCRIPT_DIR}"

if [[ "${SKIP_UI}" -eq 0 ]]; then
  require_command npm

  echo "Building UI..."
  pushd "${UI_DIR}" >/dev/null
  if [[ -f package-lock.json ]]; then
    npm ci
  else
    npm install
  fi
  npm run build
  popd >/dev/null
fi

if [[ "${SKIP_GO}" -eq 0 ]]; then
  require_command go

  mkdir -p "${BIN_DIR}"

  echo "Building Windows Go binaries..."
  pushd "${SCRIPT_DIR}" >/dev/null
  GOOS=windows GOARCH="${GOARCH_VALUE}" go build -o "${BIN_DIR}/testdataengine.exe" ./cmd/testdataengine
  GOOS=windows GOARCH="${GOARCH_VALUE}" go build -o "${BIN_DIR}/testdataengine-web.exe" ./cmd/testdataengine-web
  GOOS=windows GOARCH="${GOARCH_VALUE}" go build -o "${BIN_DIR}/csv2sqlite.exe" ./cmd/csv2sqlite
  popd >/dev/null
fi

echo
echo "Build complete."
if [[ "${SKIP_UI}" -eq 0 ]]; then
  echo "UI output: ${UI_DIR}/dist"
fi
if [[ "${SKIP_GO}" -eq 0 ]]; then
  echo "Windows binaries:"
  echo "  ${BIN_DIR}/testdataengine.exe"
  echo "  ${BIN_DIR}/testdataengine-web.exe"
  echo "  ${BIN_DIR}/csv2sqlite.exe"
fi
