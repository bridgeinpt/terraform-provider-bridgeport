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
#   scripts/acc-harness.sh down     # stop and remove the instance + its volume
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

start_container() {
  down
  log "starting ${IMAGE} on ${ENDPOINT}"
  # No volume and no DATABASE_URL override on purpose: the instance is
  # disposable, and the image's default DB path (/app/data, owned by the
  # non-root `node` user it runs as) is already writable. Mounting a fresh
  # volume at a non-default path would be root-owned and hit
  # "Permission denied (os error 13)".
  docker run -d --name "${CONTAINER}" \
    -p "${PORT}:3000" \
    -e MASTER_KEY="$(rand)" \
    -e JWT_SECRET="$(rand)" \
    -e ADMIN_EMAIL="${ADMIN_EMAIL}" \
    -e ADMIN_PASSWORD="${ADMIN_PASSWORD}" \
    "${IMAGE}" >/dev/null
}

wait_for_health() {
  log "waiting for ${ENDPOINT}/health"
  for _ in $(seq 1 90); do
    if curl -fsS "${ENDPOINT}/health" >/dev/null 2>&1; then
      log "instance healthy"
      return 0
    fi
    sleep 2
  done
  log "instance did not become healthy in time; recent logs:"
  docker logs --tail 80 "${CONTAINER}" >&2 || true
  return 1
}

mint_token() {
  # Log in as the seeded admin and extract the bearer token. Prints the token
  # to stdout; returns non-zero (and logs the response) if it can't be parsed.
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

# up starts the instance and emits endpoint+token (for CI $GITHUB_ENV and for
# local `eval "$(scripts/acc-harness.sh up)"`).
up() {
  start_container
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
  echo "export BRIDGEPORT_ENDPOINT=${ENDPOINT}"
  echo "export BRIDGEPORT_TOKEN=${token}"
}

# run_tests does the full cycle inline (no command-substitution around startup,
# so a failed health check or token mint aborts under `set -e`).
run_tests() {
  trap down EXIT
  start_container
  wait_for_health
  local token
  token="$(mint_token)"
  log "token minted; running acceptance suite (TF_ACC=1)"
  export BRIDGEPORT_ENDPOINT="${ENDPOINT}"
  export BRIDGEPORT_TOKEN="${token}"
  TF_ACC=1 go test ./internal/provider/... -v -timeout 30m
}

case "${1:-test}" in
  up)   up ;;
  down) down ;;
  test) run_tests ;;
  *)
    echo "usage: $0 {up|test|down}" >&2
    exit 2
    ;;
esac
