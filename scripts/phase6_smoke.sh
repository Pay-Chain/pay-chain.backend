#!/usr/bin/env bash
set -euo pipefail

# Phase 6 smoke test for BE rollout.
# Supports:
# 1) Admin session auth (Authorization bearer and/or X-Session-Id)
# 2) Optional payment-app call via API key signature
#
# Required:
#   BACKEND_URL, SOURCE_CHAIN_ID, DEST_CHAIN_ID
#
# Optional admin auth headers:
#   AUTH_BEARER_TOKEN
#   SESSION_ID
#   INTERNAL_PROXY_SECRET
#
# Optional payment-app signature mode:
#   RUN_PAYMENT_APP=1
#   API_KEY
#   API_SECRET
#   SENDER_WALLET_ADDRESS
#   RECEIVER_ADDRESS
#   SOURCE_TOKEN_ADDRESS
#   DEST_TOKEN_ADDRESS
#   AMOUNT
#   DECIMALS

BACKEND_URL="${BACKEND_URL:-http://localhost:8080}"
SOURCE_CHAIN_ID="${SOURCE_CHAIN_ID:-eip155:8453}"
DEST_CHAIN_ID="${DEST_CHAIN_ID:-eip155:137}"

AUTH_ARGS=()
if [[ -n "${AUTH_BEARER_TOKEN:-}" ]]; then
  AUTH_ARGS+=(-H "Authorization: Bearer ${AUTH_BEARER_TOKEN}")
fi
if [[ -n "${SESSION_ID:-}" ]]; then
  AUTH_ARGS+=(-H "X-Session-Id: ${SESSION_ID}")
fi
if [[ -n "${INTERNAL_PROXY_SECRET:-}" ]]; then
  AUTH_ARGS+=(-H "X-Internal-Proxy-Secret: ${INTERNAL_PROXY_SECRET}")
fi

request() {
  local method="$1"
  local path="$2"
  local expected="${3:-200}"
  local body="${4:-}"

  local tmp
  tmp="$(mktemp)"
  local code
  if [[ -n "${body}" ]]; then
    code="$(
      curl -sS -o "${tmp}" -w "%{http_code}" \
        -X "${method}" \
        "${BACKEND_URL}${path}" \
        "${AUTH_ARGS[@]}" \
        -H "Content-Type: application/json" \
        --data "${body}"
    )"
  else
    code="$(
      curl -sS -o "${tmp}" -w "%{http_code}" \
        -X "${method}" \
        "${BACKEND_URL}${path}" \
        "${AUTH_ARGS[@]}" \
        -H "Content-Type: application/json"
    )"
  fi

  echo "==> ${method} ${path} -> HTTP ${code}"
  cat "${tmp}"
  echo
  if [[ "${code}" != "${expected}" ]]; then
    echo "ERROR: expected HTTP ${expected}, got ${code} for ${method} ${path}" >&2
    rm -f "${tmp}"
    return 1
  fi
  rm -f "${tmp}"
}

echo "== Phase 6 smoke: admin endpoints =="
request "GET" "/api/v1/auth/me" "200"
request "GET" "/api/v1/auth/session-expiry" "200"
request "GET" "/api/v1/admin/contracts/config-check?sourceChainId=${SOURCE_CHAIN_ID}&destChainId=${DEST_CHAIN_ID}" "200"
request "GET" "/api/v1/admin/crosschain-config/preflight?sourceChainId=${SOURCE_CHAIN_ID}&destChainId=${DEST_CHAIN_ID}" "200"
request "GET" "/api/v1/admin/crosschain-config/overview?page=1&limit=20&sourceChainId=${SOURCE_CHAIN_ID}&destChainId=${DEST_CHAIN_ID}" "200"
request "POST" "/api/v1/admin/crosschain-config/recheck-bulk" "200" "{\"routes\":[{\"sourceChainId\":\"${SOURCE_CHAIN_ID}\",\"destChainId\":\"${DEST_CHAIN_ID}\"}]}"

if [[ "${RUN_PAYMENT_APP:-0}" == "1" ]]; then
  if [[ -z "${API_KEY:-}" || -z "${API_SECRET:-}" ]]; then
    echo "ERROR: RUN_PAYMENT_APP=1 requires API_KEY and API_SECRET" >&2
    exit 1
  fi

  SOURCE_TOKEN_ADDRESS="${SOURCE_TOKEN_ADDRESS:-0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913}"
  DEST_TOKEN_ADDRESS="${DEST_TOKEN_ADDRESS:-0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359}"
  AMOUNT="${AMOUNT:-100000}"
  DECIMALS="${DECIMALS:-6}"
  SENDER_WALLET_ADDRESS="${SENDER_WALLET_ADDRESS:-0xE6A7d99011257AEc28Ad60EFED58A256c4d5Fea3}"
  RECEIVER_ADDRESS="${RECEIVER_ADDRESS:-0xE6A7d99011257AEc28Ad60EFED58A256c4d5Fea3}"

  body="$(cat <<JSON
{"sourceChainId":"${SOURCE_CHAIN_ID}","destChainId":"${DEST_CHAIN_ID}","sourceTokenAddress":"${SOURCE_TOKEN_ADDRESS}","destTokenAddress":"${DEST_TOKEN_ADDRESS}","amount":"${AMOUNT}","decimals":${DECIMALS},"senderWalletAddress":"${SENDER_WALLET_ADDRESS}","receiverAddress":"${RECEIVER_ADDRESS}"}
JSON
)"
  body_hash="$(printf '%s' "${body}" | openssl dgst -sha256 -binary | xxd -p -c 256)"
  ts="$(date +%s)"
  sign_payload="${ts}POST/api/v1/payment-app${body_hash}"
  sig="$(printf '%s' "${sign_payload}" | openssl dgst -sha256 -hmac "${API_SECRET}" -binary | xxd -p -c 256)"

  echo "== Phase 6 smoke: payment-app =="
  tmp="$(mktemp)"
  code="$(
    curl -sS -o "${tmp}" -w "%{http_code}" \
      -X POST \
      "${BACKEND_URL}/api/v1/payment-app" \
      -H "Content-Type: application/json" \
      -H "X-Api-Key: ${API_KEY}" \
      -H "X-Timestamp: ${ts}" \
      -H "X-Signature: ${sig}" \
      --data "${body}"
  )"
  echo "==> POST /api/v1/payment-app -> HTTP ${code}"
  cat "${tmp}"
  echo
  if [[ "${code}" != "201" ]]; then
    echo "ERROR: expected HTTP 201 from /api/v1/payment-app, got ${code}" >&2
    rm -f "${tmp}"
    exit 1
  fi
  rm -f "${tmp}"
fi

echo "Phase 6 smoke completed successfully."
