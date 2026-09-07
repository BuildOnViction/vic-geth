#!/bin/sh
set -eu
umask 077

ROOT_DIR="/viction"
DATA_DIR="${ROOT_DIR}/datadir"
PASSWORD_FILE="${ROOT_DIR}/password_file"
PRIVKEY_FILE="${ROOT_DIR}/privkey_file"
KEYSTORE_DIR="${DATA_DIR}/keystore"
NETWORK="${NETWORK:-viction}" # available options: viction,victest
NETWORK_ID="${NETWORK_ID:-88}"
IDENTITY="${IDENTITY:-mynode}"
HTTP_API="${HTTP_API:-eth,web3}" # available options: admin, debug, eth, miner, net, personal, posv, posvdebug, txpool, web3
WS_API="${WS_API:-eth,web3}" # available options: admin, debug, eth, miner, net, personal, posv, posvdebug, txpool, web3
P2P_PORT="${P2P_PORT:-30303}"
EXTIP="${EXTIP:-}"
MAXPEERS="${MAXPEERS:-100}"
SYNCMODE="${SYNCMODE:-full}" # available options: fast, full
VERBOSITY="${VERBOSITY:-3}"
GCMODE="${GCMODE:-full}" # available options: archive, full
ETHSTATS_HOST="${ETHSTATS_HOST:-}"
ETHSTATS_PORT="${ETHSTATS_PORT:-}"
ETHSTATS_SECRET="${ETHSTATS_SECRET:-}"
PRIVKEY="${PRIVKEY:-}"
PASSWORD="${PASSWORD:-}"

# network
case "${NETWORK}" in
  viction)
    NETWORK_ID="88"
    set -- "$@" "--${NETWORK}"
    echo "Viction Mainnet network is active. Ignored custom network ID, using network 88."
    ;;
  victest)
    NETWORK_ID="89"
    set -- "$@" "--${NETWORK}"
    echo "Viction Testnet network is active. Ignored custom network ID, using network 89."
    ;;
  *)
    echo "Not supported network profile. Ignored"
    ;;
esac

# nat
if [ -n "${EXTIP}" ]; then
  set -- "$@" --nat "extip:${EXTIP}"
fi

# ethstats
if [ -n "${ETHSTATS_HOST}" ] && [ -n "${ETHSTATS_PORT}" ] && [ -n "${ETHSTATS_SECRET}" ]; then
  set -- "$@" --ethstats "${IDENTITY}:${ETHSTATS_SECRET}@${ETHSTATS_HOST}:${ETHSTATS_PORT}"
fi

# account
if [ ! -f "${PASSWORD_FILE}" ]; then
  if [ -n "${PASSWORD}" ]; then
    printf '%s\n' "${PASSWORD}" > "${PASSWORD_FILE}"
    echo "Written password env var to ${PASSWORD_FILE}"
  else
    printf '%s\n' "$(tr -dc _A-Z-a-z-0-9 < /dev/urandom | head -c 64)" > "${PASSWORD_FILE}"
    echo "Generated password to ${PASSWORD_FILE}"
  fi
fi
PRIVKEY="${PRIVKEY#0x}"
if [ -n "${PRIVKEY}" ]; then
  trap 'rm -f "${PRIVKEY_FILE}"' EXIT
  printf '%s\n' "${PRIVKEY}" > "${PRIVKEY_FILE}"
  import_rc=0
  import_log=$(/usr/local/bin/geth account import "${PRIVKEY_FILE}" --datadir "${DATA_DIR}" --keystore "${KEYSTORE_DIR}" --password "${PASSWORD_FILE}" 2>&1) || import_rc=$?
  if [ "${import_rc}" -ne 0 ]; then
    case "${import_log}" in
      *"account already exists"*)
        echo "Private key already imported, skipping import"
        ;;
      *)
        echo "${import_log}" >&2
        exit 1
        ;;
    esac
  else
    echo "Imported private key to ${KEYSTORE_DIR}"
  fi
  rm "${PRIVKEY_FILE}"
  trap - EXIT
fi
account=$(/usr/local/bin/geth account list --datadir "${DATA_DIR}" --keystore "${KEYSTORE_DIR}" 2> /dev/null | head -n 1 | cut -d"{" -f 2 | cut -d"}" -f 1)
if [ -n "${account}" ]; then
  set -- "$@" --unlock "${account}" --password "${PASSWORD_FILE}" --allow-insecure-unlock --mine
  echo "Use account ${account}"
else
  echo "No account found"
fi

exec /usr/local/bin/geth \
  --datadir "${DATA_DIR}" --networkid "${NETWORK_ID}" --identity "${IDENTITY}" \
  --http --http.addr "0.0.0.0" --http.port "8545" --http.api "${HTTP_API}" --http.corsdomain "*" --http.vhosts "*" \
  --ws --ws.addr "0.0.0.0" --ws.port "8546" --ws.api "${WS_API}" --ws.origins "*" \
  --port "${P2P_PORT}" --maxpeers "${MAXPEERS}" --syncmode "${SYNCMODE}" --cache.noprefetch \
  --ipcpath "${DATA_DIR}/vic-geth.ipc" --verbosity "${VERBOSITY}" --gcmode "${GCMODE}" \
  "$@"
