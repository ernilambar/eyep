#!/usr/bin/env bats

bats_require_minimum_version 1.5.0

setup() {
  SCRIPT="$BATS_TEST_DIRNAME/../eyep.sh"
  STUBS="$BATS_TEST_DIRNAME/stubs"
  PATH="$STUBS:$PATH"
  CURL_LOG="$BATS_TEST_TMPDIR/curl.log"
  EMPTY_BIN="$BATS_TEST_TMPDIR/empty-bin"
  mkdir -p "$EMPTY_BIN"
  export PATH CURL_LOG
}

#
# Help and version
#

@test "-h and --help print usage on stdout and exit 0" {
  for flag in -h --help; do
    run --separate-stderr "$SCRIPT" "$flag"
    [ "$status" -eq 0 ]
    [ -z "$stderr" ]
    [[ "$output" == *"Usage:"* ]]
  done
}

@test "-V and --version print a semver version and exit 0" {
  for flag in -V --version; do
    run "$SCRIPT" "$flag"
    [ "$status" -eq 0 ]
    [[ "$output" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]
  done
}

@test "help inside a short-flag cluster exits 0" {
  run --separate-stderr "$SCRIPT" -ih
  [ "$status" -eq 0 ]
  [ -z "$stderr" ]
  [[ "$output" == *"Usage:"* ]]
}

#
# Flag parsing errors
#

@test "unknown long flag errors on stderr and exits 1" {
  run --separate-stderr "$SCRIPT" --bogus
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Unknown flag --bogus"* ]]
}

@test "unknown short flag errors on stderr and exits 1" {
  run --separate-stderr "$SCRIPT" -z
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Unknown flag -z"* ]]
}

@test "unknown flag inside a cluster errors on stderr and exits 1" {
  run --separate-stderr "$SCRIPT" -iz
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Unknown flag -z"* ]]
}

@test "extra positional argument errors on stderr and exits 1" {
  run --separate-stderr "$SCRIPT" 1.2.3.4 5.6.7.8
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Unexpected argument"* ]]
}

#
# IP address validation
#

@test "invalid IPv4 out-of-range octet errors and exits 1" {
  run --separate-stderr "$SCRIPT" 999.1.1.1
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Invalid IP address: 999.1.1.1"* ]]
}

@test "non-IP argument errors and exits 1" {
  run --separate-stderr "$SCRIPT" notanip
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Invalid IP address: notanip"* ]]
}

@test "invalid IP is rejected even in ip-only mode, with no network call" {
  run --separate-stderr "$SCRIPT" -i notanip
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"Invalid IP address: notanip"* ]]
  [ ! -e "$CURL_LOG" ]
}

@test "IPv4 octet boundaries are accepted" {
  for ip in 0.0.0.0 255.255.255.255; do
    run "$SCRIPT" -ij "$ip"
    [ "$status" -eq 0 ]
    [ "$output" = "{\"ip\":\"$ip\"}" ]
  done
}

@test "malformed IPv4 addresses are rejected" {
  for ip in 1.2.3 1.2.3.4.5 256.0.0.1 1.2.3.x; do
    run --separate-stderr "$SCRIPT" "$ip"
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"Invalid IP address: $ip"* ]]
  done
}

@test "valid IPv6 address is accepted" {
  CURL_STUB_JSON='{"ipAddress":"2001:4860:4860::8888"}' \
    run "$SCRIPT" -j 2001:4860:4860::8888
  [ "$status" -eq 0 ]
  [ "$output" = '{"ipAddress":"2001:4860:4860::8888"}' ]
}

@test "IPv6 loopback and compressed forms are accepted" {
  for ip in ::1 2001:db8::1 fe80::1; do
    run "$SCRIPT" -ij "$ip"
    [ "$status" -eq 0 ]
    [ "$output" = "{\"ip\":\"$ip\"}" ]
  done
}

@test "malformed IPv6 addresses are rejected" {
  for ip in : ::: :::: 1::2::3 :1:2 1:2: gggg::1; do
    run --separate-stderr "$SCRIPT" "$ip"
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"Invalid IP address: $ip"* ]]
  done
}

#
# Argument separators
#

@test "-- terminates option parsing and treats the next argument as the IP" {
  run "$SCRIPT" -ij -- 8.8.8.8
  [ "$status" -eq 0 ]
  [ "$output" = '{"ip":"8.8.8.8"}' ]
}

@test "option-like argument after -- is treated as the IP and rejected" {
  run --separate-stderr "$SCRIPT" -i -- -h
  [ "$status" -eq 1 ]
  [[ "$stderr" == *"Invalid IP address: -h"* ]]
}

#
# Human-readable output
#

@test "human-readable output shows N/A for missing fields" {
  CURL_STUB_JSON='{"status":"success","query":"8.8.8.8"}' run "$SCRIPT" 8.8.8.8
  [ "$status" -eq 0 ]
  [[ "$output" == *"Country: N/A (N/A)"* ]]
  [[ "$output" == *"Region: N/A"* ]]
}

@test "human-readable output renders the full key-value table" {
  CURL_STUB_JSON='{"status":"success","query":"8.8.8.8","country":"United States","countryCode":"US","regionName":"California","city":"Mountain View","zip":"94043","lat":37.4,"lon":-122.1,"timezone":"America/Los_Angeles","isp":"Google LLC","org":"Google Public DNS","as":"AS15169 Google LLC"}' \
    run "$SCRIPT" 8.8.8.8
  [ "$status" -eq 0 ]
  [[ "$output" == *"IP: 8.8.8.8"* ]]
  [[ "$output" == *"Country: United States (US)"* ]]
  [[ "$output" == *"Region: California"* ]]
  [[ "$output" == *"City: Mountain View 94043"* ]]
  [[ "$output" == *"Coordinates: 37.4, -122.1"* ]]
  [[ "$output" == *"Timezone: America/Los_Angeles"* ]]
  [[ "$output" == *"ISP: Google LLC"* ]]
  [[ "$output" == *"Org: Google Public DNS"* ]]
  [[ "$output" == *"AS: AS15169 Google LLC"* ]]
}

@test "city renders without a trailing zip when zip is absent" {
  CURL_STUB_JSON='{"status":"success","query":"8.8.8.8","city":"Mountain View"}' \
    run "$SCRIPT" 8.8.8.8
  [ "$status" -eq 0 ]
  [[ "$output" == *"City: Mountain View"* ]]
  [[ "$output" != *"City: Mountain View "* ]]
}

#
# JSON output
#

@test "-j with explicit IP prints raw JSON from the API" {
  CURL_STUB_JSON='{"ipAddress":"8.8.8.8","countryName":"United States"}' \
    run "$SCRIPT" -j 8.8.8.8
  [ "$status" -eq 0 ]
  [ "$output" = '{"ipAddress":"8.8.8.8","countryName":"United States"}' ]
}

@test "-j with no IP discovers own IP then prints raw JSON" {
  CURL_STUB_IP="198.51.100.7" \
    CURL_STUB_JSON='{"status":"success","query":"198.51.100.7"}' \
    run "$SCRIPT" -j
  [ "$status" -eq 0 ]
  [ "$output" = '{"status":"success","query":"198.51.100.7"}' ]
}

#
# IP-only output
#

@test "-i with explicit IP prints only the IP, no network call" {
  run "$SCRIPT" -i 8.8.8.8
  [ "$status" -eq 0 ]
  [ "$output" = "8.8.8.8" ]
  [ ! -e "$CURL_LOG" ]
}

@test "-ij with explicit IP prints IP-only JSON, no network call" {
  run "$SCRIPT" -ij 8.8.8.8
  [ "$status" -eq 0 ]
  [ "$output" = '{"ip":"8.8.8.8"}' ]
  [ ! -e "$CURL_LOG" ]
}

@test "-ij with no IP discovers IP and skips the lookup call" {
  CURL_STUB_IP="203.0.113.9" run "$SCRIPT" -ij
  [ "$status" -eq 0 ]
  [ "$output" = '{"ip":"203.0.113.9"}' ]
  grep -q 'api64.ipify.org' "$CURL_LOG"
  ! grep -q 'ip-api.com' "$CURL_LOG"
}

#
# Address-family flags
#

@test "combined short flags -ij4 force IPv4 for IP discovery" {
  CURL_STUB_IP="203.0.113.9" run "$SCRIPT" -ij4
  [ "$status" -eq 0 ]
  [ "$output" = '{"ip":"203.0.113.9"}' ]
  grep -q -- '-4' "$CURL_LOG"
}

@test "combined short flags -ij6 force IPv6 for IP discovery" {
  CURL_STUB_IP="2001:db8::9" run "$SCRIPT" -ij6
  [ "$status" -eq 0 ]
  [ "$output" = '{"ip":"2001:db8::9"}' ]
  grep -q -- '-6' "$CURL_LOG"
}

@test "--ipv4 forces IPv4 for IP discovery" {
  CURL_STUB_IP="203.0.113.9" run "$SCRIPT" --ipv4 -i
  [ "$status" -eq 0 ]
  [ "$output" = "203.0.113.9" ]
  grep -q -- '-4' "$CURL_LOG"
}

@test "--ipv6 forces IPv6 for IP discovery" {
  CURL_STUB_IP="2001:db8::9" run "$SCRIPT" --ipv6 -i
  [ "$status" -eq 0 ]
  [ "$output" = "2001:db8::9" ]
  grep -q -- '-6' "$CURL_LOG"
}

#
# Discovery and lookup failures
#

@test "discovers own IP via curl when no IP given" {
  CURL_STUB_IP="198.51.100.7" run "$SCRIPT" -i
  [ "$status" -eq 0 ]
  [ "$output" = "198.51.100.7" ]
}

@test "errors on stderr when IP discovery fails" {
  CURL_STUB_IP_EXIT=1 run --separate-stderr "$SCRIPT" -i
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Could not determine public IP address"* ]]
}

@test "errors when the lookup fails after a successful discovery" {
  CURL_STUB_IP="203.0.113.9" CURL_STUB_JSON_EXIT=1 \
    run --separate-stderr "$SCRIPT" -j
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Failed to fetch data for IP 203.0.113.9"* ]]
}

@test "errors when the lookup fails for an explicit IP" {
  CURL_STUB_EXIT=1 run --separate-stderr "$SCRIPT" 8.8.8.8
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Failed to fetch data for IP 8.8.8.8"* ]]
}

#
# Malformed API responses
#

@test "non-JSON API response errors cleanly" {
  CURL_STUB_JSON='<html>rate limited</html>' run --separate-stderr "$SCRIPT" 8.8.8.8
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Unexpected response from API for IP 8.8.8.8"* ]]
}

@test "API status=fail surfaces the API message and exits 1" {
  CURL_STUB_JSON='{"status":"fail","message":"invalid query"}' \
    run --separate-stderr "$SCRIPT" 8.8.8.8
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Error: invalid query"* ]]
}

@test "API response without a success status uses the default error message" {
  CURL_STUB_JSON='{}' run --separate-stderr "$SCRIPT" 8.8.8.8
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Invalid IP address or API lookup error."* ]]
}

#
# Request shape
#

@test "lookup requests the target IP and the expected fields" {
  CURL_STUB_JSON='{"status":"success","query":"8.8.8.8"}' run "$SCRIPT" 8.8.8.8
  [ "$status" -eq 0 ]
  grep -q 'ip-api.com/json/8.8.8.8' "$CURL_LOG"
  grep -q 'fields=status,message,country,countryCode' "$CURL_LOG"
}

#
# External dependencies
#

@test "errors when curl is unavailable" {
  run --separate-stderr env PATH="$EMPTY_BIN" "$SCRIPT" -j 8.8.8.8
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Required command not found: curl"* ]]
}

@test "errors when jq is unavailable for table output" {
  run --separate-stderr env CURL_STUB_JSON='{"status":"success","query":"8.8.8.8"}' \
    PATH="$STUBS" "$SCRIPT" 8.8.8.8
  [ "$status" -eq 1 ]
  [ -z "$output" ]
  [[ "$stderr" == *"Required command not found: jq"* ]]
}

#
# Static checks
#

@test "script passes the sh -n syntax check" {
  run sh -n "$SCRIPT"
  [ "$status" -eq 0 ]
}

@test "shellcheck reports no issues" {
  command -v shellcheck >/dev/null 2>&1 || skip "shellcheck not installed"
  run shellcheck "$SCRIPT"
  [ "$status" -eq 0 ]
}
