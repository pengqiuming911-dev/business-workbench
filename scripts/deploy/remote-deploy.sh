#!/usr/bin/env bash

set -euo pipefail

APP_DIR="${APP_DIR:-/var/www/business-workbench}"
SERVICE_NAME="${SERVICE_NAME:-business-workbench}"
PORT="${PORT:-3001}"
RELEASE_NAME="${RELEASE_NAME:-release}"
RELEASE_TARBALL="${RELEASE_TARBALL:-/tmp/${RELEASE_NAME}.tar.gz}"
RELEASES_DIR="${APP_DIR}/releases"
SHARED_DIR="${APP_DIR}/shared"
CURRENT_LINK="${APP_DIR}/current"
ENV_FILE="${SHARED_DIR}/.env"
DB_FILE="${SHARED_DIR}/data.sqlite"
PREVIOUS_TARGET=""

if [ ! -f "${RELEASE_TARBALL}" ]; then
  echo "Release tarball not found: ${RELEASE_TARBALL}" >&2
  exit 1
fi

mkdir -p "${RELEASES_DIR}" "${SHARED_DIR}"

if [ -L "${CURRENT_LINK}" ]; then
  PREVIOUS_TARGET="$(readlink -f "${CURRENT_LINK}" || true)"
fi

RELEASE_DIR="${RELEASES_DIR}/${RELEASE_NAME}"
rm -rf "${RELEASE_DIR}"
mkdir -p "${RELEASE_DIR}"

tar -xzf "${RELEASE_TARBALL}" -C "${RELEASE_DIR}"

mkdir -p "${RELEASE_DIR}/backend-go"
mkdir -p "${RELEASE_DIR}/frontend"

if [ ! -f "${ENV_FILE}" ]; then
  echo "Shared env file missing: ${ENV_FILE}" >&2
  exit 1
fi

ln -sfn "${ENV_FILE}" "${RELEASE_DIR}/backend-go/.env"

if [ -f "${DB_FILE}" ]; then
  ln -sfn "${DB_FILE}" "${RELEASE_DIR}/backend-go/data.sqlite"
fi

chmod +x "${RELEASE_DIR}/server"
ln -sfn "${RELEASE_DIR}" "${CURRENT_LINK}"

ln -sfn "${CURRENT_LINK}/frontend" "${APP_DIR}/frontend"
ln -sfn "${CURRENT_LINK}/backend-go" "${APP_DIR}/backend-go"

sudo tee "/etc/systemd/system/${SERVICE_NAME}.service" >/dev/null <<EOF
[Unit]
Description=Business Workbench Go Backend
After=network.target

[Service]
Type=simple
User=$(id -un)
WorkingDirectory=${CURRENT_LINK}/backend-go
ExecStart=${CURRENT_LINK}/server
Restart=on-failure
RestartSec=5
Environment=PORT=${PORT}

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable "${SERVICE_NAME}" >/dev/null 2>&1 || true
sudo systemctl restart "${SERVICE_NAME}"

for _ in $(seq 1 15); do
  if curl -fsS "http://127.0.0.1:${PORT}/api/health" >/dev/null; then
    sudo systemctl --no-pager --full status "${SERVICE_NAME}" || true
    rm -f "${RELEASE_TARBALL}"
    exit 0
  fi
  sleep 2
done

echo "Health check failed for ${SERVICE_NAME}" >&2
sudo journalctl -u "${SERVICE_NAME}" -n 100 --no-pager || true

if [ -n "${PREVIOUS_TARGET}" ] && [ -d "${PREVIOUS_TARGET}" ]; then
  ln -sfn "${PREVIOUS_TARGET}" "${CURRENT_LINK}"
  ln -sfn "${CURRENT_LINK}/frontend" "${APP_DIR}/frontend"
  ln -sfn "${CURRENT_LINK}/backend-go" "${APP_DIR}/backend-go"
  sudo systemctl restart "${SERVICE_NAME}" || true
fi

exit 1
