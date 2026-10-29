#!/bin/sh
set -eu

VERSION="1.1.0"

# Network timeouts (seconds) so an unresponsive endpoint cannot hang the CLI.
CONNECT_TIMEOUT=5
MAX_TIME=10

usage() {
  status="${1:-0}"
  # Explicit help (-h/--help) goes to stdout; usage printed on an error goes to stderr.
  fd=1
  [ "$status" -eq 0 ] || fd=2
  cat <<EOF >&$fd
Usage: ${0##*/} [FLAGS] [IP_ADDRESS]

Inspect geographical and network details for an IP address or your local machine.

Flags:
  -i, --ip-only    Print only the IP address (concise mode)
  -j, --json       Output raw structured JSON
  -4, --ipv4       Force IPv4 network connection (default)
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
  exit "$status"
}

# Abort with a clear message if a required command is missing.
require() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'Error: Required command not found: %s\n' "$1" >&2
    exit 1
  }
}

# Return 0 if the argument is a valid dotted-decimal IPv4 address.
is_valid_ipv4() {
  case "$1" in
    '' | *[!0-9.]*) return 1 ;;
    *.*.*.*) ;;
    *) return 1 ;;
  esac
  # Exactly four dot-separated octets, each in range 0-255.
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
}

# Count the colon-separated hex groups in an IPv6 fragment. Sets GROUP_COUNT
# and returns non-zero if any group is empty or longer than four hex digits.
count_groups() {
  GROUP_COUNT=0
  case "$1" in
    '') return 0 ;;
    :* | *:) return 1 ;;
  esac
  rest="$1"
  while [ -n "$rest" ]; do
    case "$rest" in
      *:*) group="${rest%%:*}" ; rest="${rest#*:}" ;;
      *) group="$rest" ; rest="" ;;
    esac
    case "$group" in
      '' | *[!0-9A-Fa-f]*) return 1 ;;
    esac
    [ "${#group}" -le 4 ] || return 1
    GROUP_COUNT=$((GROUP_COUNT + 1))
  done
  return 0
}

# Return 0 if the argument is a valid IPv6 address, including forms with an
# embedded IPv4 tail such as ::ffff:192.168.0.1.
is_valid_ipv6() {
  addr="$1"

  # An embedded IPv4 tail stands for the last two hex groups.
  case "$addr" in
    *.*)
      case "$addr" in
        *:*) ;;
        *) return 1 ;;
      esac
      is_valid_ipv4 "${addr##*:}" || return 1
      addr="${addr%:*}:0:0"
      ;;
  esac

  # Only hex digits and colons may remain.
  case "$addr" in
    '' | *[!0-9A-Fa-f:]*) return 1 ;;
  esac
  # ":::" is never valid.
  case "$addr" in
    *:::*) return 1 ;;
  esac

  case "$addr" in
    *::*)
      count_groups "${addr%%::*}" || return 1
      left="$GROUP_COUNT"
      count_groups "${addr#*::}" || return 1
      # "::" must stand in for at least one zero group.
      [ $((left + GROUP_COUNT)) -le 7 ] || return 1
      ;;
    *)
      count_groups "$addr" || return 1
      [ "$GROUP_COUNT" -eq 8 ] || return 1
      ;;
  esac
}

# Return 0 if the argument is a valid IPv4 or IPv6 address.
is_valid_ip() {
  # Colons first: IPv4-mapped IPv6 addresses also contain dots.
  case "$1" in
    *:*) is_valid_ipv6 "$1" ;;
    *.*) is_valid_ipv4 "$1" ;;
    *) return 1 ;;
  esac
}

eyep() {
  IP_ONLY=0
  JSON_OUTPUT=0
  # Default to IPv4; -4/-6 override this.
  CURL_IP_FLAGS="-4"
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
    TARGET_IP=$(curl -fsSL --connect-timeout "$CONNECT_TIMEOUT" --max-time "$MAX_TIME" $CURL_IP_FLAGS "https://api64.ipify.org" 2>/dev/null) || true
    # Only trust the response if it really is an IP address; it is embedded in
    # JSON output and used to build the lookup URL.
    if [ -z "$TARGET_IP" ] || ! is_valid_ip "$TARGET_IP"; then
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
  # Table output needs jq; fail before spending a network request.
  [ "$JSON_OUTPUT" -eq 1 ] || require jq
  data=$(curl -fsSL --connect-timeout "$CONNECT_TIMEOUT" --max-time "$MAX_TIME" $CURL_IP_FLAGS "http://ip-api.com/json/${TARGET_IP}?fields=status,message,country,countryCode,regionName,city,zip,lat,lon,timezone,isp,org,as,query" 2>/dev/null) || true
  if [ -z "$data" ]; then
    printf 'Error: Failed to fetch data for IP %s.\n' "$TARGET_IP" >&2
    return 1
  fi

  # Step 4: Render JSON or Key-Value output
  if [ "$JSON_OUTPUT" -eq 1 ]; then
    printf '%s\n' "$data"
  else
    if ! table=$(printf '%s' "$data" | jq -r '
      if .status == "success" then
        [
          ["IP", (.query // "N/A")],
          ["Country", "\(.country // "N/A") (\(.countryCode // "N/A"))"],
          ["Region", (.regionName // "N/A" | if . == "" then "N/A" else . end)],
          ["City", ((.city // "N/A" | if . == "" then "N/A" else . end) + " " + (.zip // "")) | rtrimstr(" ") | if . == "" then "N/A" else . end],
          ["Coordinates", "\(.lat // "N/A"), \(.lon // "N/A")"],
          ["Timezone", (.timezone // "N/A" | if . == "" then "N/A" else . end)],
          ["ISP", (.isp // "N/A" | if . == "" then "N/A" else . end)],
          ["Org", (.org // "N/A" | if . == "" then "N/A" else . end)],
          ["AS", (.as // "N/A" | if . == "" then "N/A" else . end)]
        ] | .[] | "\(.[0]): \(.[1])"
      else
        "Error: \(.message // "Invalid IP address or API lookup error.")"
      end
    ' 2>/dev/null); then
      printf 'Error: Unexpected response from API for IP %s.\n' "$TARGET_IP" >&2
      return 1
    fi
    case "$table" in
      Error:*)
        printf '%s\n' "$table" >&2
        return 1
        ;;
    esac
    printf '%s\n' "$table"
  fi
}

eyep "$@"
