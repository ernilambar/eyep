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
      --version    Display version information

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

  # Step 1: Determine target IP if omitted
  if [ -z "$TARGET_IP" ]; then
    TARGET_IP=$(curl -fsSL $CURL_IP_FLAGS "https://api64.ipify.org" 2>/dev/null)
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
  data=$(curl -fsSL $CURL_IP_FLAGS "https://freeipapi.com/api/json/${TARGET_IP}" 2>/dev/null)
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
    printf '%s' "$data" | jq -r '
      if .ipAddress then
        [
          ["IP", .ipAddress],
          ["IP Version", (if .ipVersion then .ipVersion | tostring else "N/A" end)],
          ["Country", "\(.countryName) (\(.countryCode))"],
          ["Region", .regionName],
          ["City", "\(.cityName) \(.zipCode)"],
          ["Coordinates", "\(.latitude), \(.longitude)"],
          ["Timezone", (.timeZones | join(", "))]
        ] | .[] | "\(.[0]): \(.[1])"
      else
        "Error: Invalid IP address or API lookup error."
      end
    '
  fi
}

eyep "$@"
