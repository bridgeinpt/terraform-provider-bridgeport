#!/usr/bin/env bash
#
# acc-harness.sh — disposable BridgePort instance for acceptance tests.
#
# A fresh BridgePort is cheap: SQLite + the ADMIN_EMAIL/ADMIN_PASSWORD first-boot
# bootstrap mean we can `docker run` an instance, mint a token by logging in as
# the seeded admin, run the TF_ACC suite against it, and throw it away.
#
# Usage:
#   scripts/acc-harness.sh up      # start instance, print endpoint+token (export form)
#   scripts/acc-harness.sh test    # up -> go test (TF_ACC=1) -> down
#   scripts/acc-harness.sh down     # stop and remove the instance
#
# Environment:
#   BRIDGEPORT_IMAGE   image to test (default: ghcr.io/bridgeinpt/bridgeport:edge)
#   BRIDGEPORT_PORT    host port to bind (default: 3000)
#   GITHUB_ENV         if set (CI), endpoint+token are appended for later steps
#
set -euo pipefail

IMAGE="${BRIDGEPORT_IMAGE:-ghcr.io/bridgeinpt/bridgeport:edge}"
PORT="${BRIDGEPORT_PORT:-3000}"
CONTAINER="bridgeport-acc"
ADMIN_EMAIL="acc-admin@bridgeport.test"
ADMIN_PASSWORD="acc-$(date +%s)-$$"
ENDPOINT="http://127.0.0.1:${PORT}"

log() { printf '\033[36m[acc-harness]\033[0m %s\n' "$*" >&2; }

rand() { openssl rand -base64 32; }

down() {
  log "tearing down ${CONTAINER}"
  docker rm -f "${CONTAINER}" >/dev/null 2>&1 || true
}

wait_for_health() {
  log "waiting for ${ENDPOINT}/health"
  for _ in $(seq 1 60); do
    if curl -fsS "${ENDPOINT}/health" >/dev/null 2>&1; then
      log "instance healthy"
      return 0
    fi
    sleep 2
  done
  log "instance did not become healthy in time; recent logs:"
  docker logs --tail 50 "${CONTAINER}" >&2 || true
  return 1
}

mint_token() {
  # Log in as the seeded admin and extract the bearer token from the response.
  local resp token
  resp="$(curl -fsS -X POST "${ENDPOINT}/api/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"${ADMIN_EMAIL}\",\"password\":\"${ADMIN_PASSWORD}\"}")"
  token="$(printf '%s' "${resp}" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')"
  if [ -z "${token}" ]; then
    log "failed to mint token; login response: ${resp}"
    return 1
  fi
  printf '%s' "${token}"
}

up() {
  down
  log "starting ${IMAGE} on ${ENDPOINT}"
  docker run -d --name "${CONTAINER}" \
    -p "${PORT}:3000" \
    -e DATABASE_URL="file:/data/bridgeport.db" \
    -e MASTER_KEY="$(rand)" \
    -e JWT_SECRET="$(rand)" \
    -e ADMIN_EMAIL="${ADMIN_EMAIL}" \
    -e ADMIN_PASSWORD="${ADMIN_PASSWORD}" \
    "${IMAGE}" >/dev/null

  wait_for_health
  local token
  token="$(mint_token)"
  log "token minted"

  if [ -n "${GITHUB_ENV:-}" ]; then
    {
      echo "BRIDGEPORT_ENDPOINT=${ENDPOINT}"
      echo "BRIDGEPORT_TOKEN=${token}"
    } >>"${GITHUB_ENV}"
  fi

  # Also emit shell-eval'able exports for local use: eval "$(acc-harness.sh up)"
  echo "export BRIDGEPORT_ENDPOINT=${ENDPOINT}"
  echo "export BRIDGEPORT_TOKEN=${token}"
}

case "${1:-test}" in
  up)
    up
    ;;
  down)
    down
    ;;
  test)
    trap down EXIT
    eval "$(up)"
    log "running acceptance suite (TF_ACC=1)"
    TF_ACC=1 go test ./internal/provider/... -v -timeout 30m
    ;;
  *)
    echo "usage: $0 {up|test|down}" >&2
    exit 2
    ;;
esac
