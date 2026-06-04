#!/usr/bin/env sh
set -eu

APP_NAME="api"
REMOVED=0

for target in "/usr/local/bin/${APP_NAME}" "${HOME}/bin/${APP_NAME}"; do
  if [ -f "${target}" ]; then
    rm -f "${target}"
    echo "Removed ${target}"
    REMOVED=1
  fi
done

if [ "${REMOVED}" -eq 0 ]; then
  echo "${APP_NAME} was not found in /usr/local/bin or ~/bin"
fi
