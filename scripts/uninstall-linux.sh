#!/usr/bin/env sh
set -eu

APP_NAME="api"
TARGET="${HOME}/.local/bin/${APP_NAME}"

if [ -f "${TARGET}" ]; then
  rm -f "${TARGET}"
  echo "Removed ${TARGET}"
else
  echo "${APP_NAME} is not installed at ${TARGET}"
fi
