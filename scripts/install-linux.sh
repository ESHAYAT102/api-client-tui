#!/usr/bin/env sh
set -eu

APP_NAME="api"
ROOT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
INSTALL_DIR="${HOME}/.local/bin"
TARGET="${INSTALL_DIR}/${APP_NAME}"

command -v go >/dev/null 2>&1 || {
  echo "go is required but was not found in PATH" >&2
  exit 1
}

mkdir -p "${INSTALL_DIR}"
cd "${ROOT_DIR}"
go build -buildvcs=false -o "${TARGET}" .
chmod 0755 "${TARGET}"

echo "Installed ${APP_NAME} to ${TARGET}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "Add ${INSTALL_DIR} to PATH to run '${APP_NAME}' from anywhere." ;;
esac
