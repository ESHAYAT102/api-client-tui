#!/usr/bin/env sh
set -eu

APP_NAME="api"
REPO_URL="https://github.com/ESHAYAT102/api-client-tui.git"
CLONE_DIR=$(mktemp -d "${TMPDIR:-/tmp}/${APP_NAME}.install.XXXXXX")

cleanup() {
  rm -rf "${CLONE_DIR}"
}
trap cleanup EXIT INT HUP TERM

command -v go >/dev/null 2>&1 || {
  echo "go is required but was not found in PATH" >&2
  exit 1
}

command -v git >/dev/null 2>&1 || {
  echo "git is required but was not found in PATH" >&2
  exit 1
}

if [ -w "/usr/local/bin" ]; then
  INSTALL_DIR="/usr/local/bin"
else
  INSTALL_DIR="${HOME}/bin"
fi

TARGET="${INSTALL_DIR}/${APP_NAME}"
mkdir -p "${INSTALL_DIR}"
git clone --depth 1 "${REPO_URL}" "${CLONE_DIR}"
cd "${CLONE_DIR}"
go build -buildvcs=false -o "${TARGET}" .
chmod 0755 "${TARGET}"

echo "Installed ${APP_NAME} to ${TARGET}"
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "Add ${INSTALL_DIR} to PATH to run '${APP_NAME}' from anywhere." ;;
esac
