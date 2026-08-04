#!/bin/sh
set -eu

VERSION="1.0.0"

usage() {
  cat <<EOF
Usage: ${0##*/} [FLAGS] [IP_ADDRESS]

Inspect geographical and network details for an IP address or your local machine.

Flags:
  -i, --ip-only    Print only the IP address (concise mode)
  -j, --json       Output raw structured JSON
  -4, --ipv4       Force IPv4 network connection
  -6, --ipv6       Force IPv6 network connection
  -h, --help       Display this help message
  -V, --version    Display version information

Short flags may be combined, e.g. -ij4.

Examples:
  eyep                    # Human-readable table
  eyep -j                 # Raw JSON output for local machine
  eyep -j 8.8.8.8         # Raw JSON output for target IP
  eyep -j 8.8.8.8 | jq .  # Pretty-printed JSON
  eyep -ij4               # IP-only, JSON, forced over IPv4
EOF
  exit "${1:-0}"
}

# Abort with a clear message if a required command is missing.
require() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'Error: Required command not found: %s\n' "$1" >&2
    exit 1
  }
}

# Return 0 if the argument looks like a valid IPv4 or IPv6 address.
is_valid_ip() {
  case "$1" in
    *.*.*.*)
      # IPv4: four dot-separated decimal octets in range 0-255.
      ( IFS=.
        # shellcheck disable=SC2086
        set -- $1
        [ $# -eq 4 ] || exit 1
        for octet; do
          case "$octet" in
            '' | *[!0-9]*) exit 1 ;;
          esac
          [ "$octet" -le 255 ] || exit 1
        done )
      ;;
    *:*)
      # IPv6: hex digits and colons only.
      case "$1" in
        *[!0-9A-Fa-f:]*) return 1 ;;
      esac
      ;;
    *)
      return 1
      ;;
  esac
}

eyep() {
  IP_ONLY=0
  JSON_OUTPUT=0
  CURL_IP_FLAGS=""
  TARGET_IP=""

  # POSIX argument and flag parser
  while [ $# -gt 0 ]; do
    case "$1" in
      --)
        shift
        break
        ;;
      --ip-only)
        IP_ONLY=1
        shift
        ;;
      --json)
        JSON_OUTPUT=1
        shift
        ;;
      --ipv4)
        CURL_IP_FLAGS="-4"
        shift
        ;;
      --ipv6)
        CURL_IP_FLAGS="-6"
        shift
        ;;
      --help)
        usage 0
        ;;
      --version)
        printf '%s\n' "$VERSION"
        exit 0
        ;;
      --*)
        printf 'Error: Unknown flag %s\n\n' "$1" >&2
        usage 1
        ;;
      -?*)
        # Cluster of short flags, e.g. -ij4
        flags="${1#-}"
        shift
        while [ -n "$flags" ]; do
          flag="${flags%"${flags#?}"}"
          flags="${flags#?}"
          case "$flag" in
            i) IP_ONLY=1 ;;
            j) JSON_OUTPUT=1 ;;
            4) CURL_IP_FLAGS="-4" ;;
            6) CURL_IP_FLAGS="-6" ;;
            h) usage 0 ;;
            V)
              printf '%s\n' "$VERSION"
              exit 0
              ;;
            *)
              printf 'Error: Unknown flag -%s\n\n' "$flag" >&2
              usage 1
              ;;
          esac
        done
        ;;
      *)
        if [ -z "$TARGET_IP" ]; then
          TARGET_IP="$1"
        else
          printf 'Error: Unexpected argument %s\n\n' "$1" >&2
          usage 1
        fi
        shift
        ;;
    esac
  done

  # Anything left after -- is positional (the target IP)
  if [ $# -gt 0 ]; then
    if [ -z "$TARGET_IP" ]; then
      TARGET_IP="$1"
      shift
    fi
    if [ $# -gt 0 ]; then
      printf 'Error: Unexpected argument %s\n\n' "$1" >&2
      usage 1
    fi
  fi

  # Validate an explicitly provided target IP.
  if [ -n "$TARGET_IP" ] && ! is_valid_ip "$TARGET_IP"; then
    printf 'Error: Invalid IP address: %s\n' "$TARGET_IP" >&2
    return 1
  fi

  # Step 1: Determine target IP if omitted
  if [ -z "$TARGET_IP" ]; then
    require curl
    TARGET_IP=$(curl -fsSL $CURL_IP_FLAGS "https://api64.ipify.org" 2>/dev/null) || true
    if [ -z "$TARGET_IP" ]; then
      printf 'Error: Could not determine public IP address.\n' >&2
      return 1
    fi
  fi

  # Step 2: Handle IP-only output (-i takes precedence)
  if [ "$IP_ONLY" -eq 1 ]; then
    if [ "$JSON_OUTPUT" -eq 1 ]; then
      printf '{"ip":"%s"}\n' "$TARGET_IP"
    else
      printf '%s\n' "$TARGET_IP"
    fi
    return 0
  fi

  # Step 3: Fetch API response
  require curl
  data=$(curl -fsSL $CURL_IP_FLAGS "https://freeipapi.com/api/json/${TARGET_IP}" 2>/dev/null) || true
  if [ -z "$data" ]; then
    printf 'Error: Failed to fetch data for IP %s.\n' "$TARGET_IP" >&2
    return 1
  fi

  # Step 4: Render JSON or Key-Value output
  if [ "$JSON_OUTPUT" -eq 1 ]; then
    # Output raw JSON directly (or minify/validate with jq if preferred)
    printf '%s\n' "$data"
  else
    # Output human-readable key-value table
    require jq
    if ! table=$(printf '%s' "$data" | jq -r '
      if .ipAddress then
        [
          ["IP", .ipAddress],
          ["IP Version", (.ipVersion // "N/A" | tostring)],
          ["Country", "\(.countryName // "N/A") (\(.countryCode // "N/A"))"],
          ["Region", (.regionName // "N/A")],
          ["City", "\(.cityName // "N/A") \(.zipCode // "")"],
          ["Coordinates", "\(.latitude // "N/A"), \(.longitude // "N/A")"],
          ["Timezone", ((.timeZones // []) | join(", "))]
        ] | .[] | "\(.[0]): \(.[1])"
      else
        "Error: Invalid IP address or API lookup error."
      end
    ' 2>/dev/null); then
      printf 'Error: Unexpected response from API for IP %s.\n' "$TARGET_IP" >&2
      return 1
    fi
    printf '%s\n' "$table"
  fi
}

eyep "$@"
