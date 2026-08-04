#!/usr/bin/env bats

setup() {
  SCRIPT="$BATS_TEST_DIRNAME/../eyep.sh"
  STUBS="$BATS_TEST_DIRNAME/stubs"
  PATH="$STUBS:$PATH"
  CURL_LOG="$BATS_TEST_TMPDIR/curl.log"
  export PATH CURL_LOG
}

@test "-h prints usage and exits 0" {
  run "$SCRIPT" -h
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage:"* ]]
}

@test "--help prints usage and exits 0" {
  run "$SCRIPT" --help
  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage:"* ]]
}

@test "-V prints version and exits 0" {
  expected=$(sed -n 's/^VERSION="\(.*\)"/\1/p' "$SCRIPT")
  run "$SCRIPT" -V
  [ "$status" -eq 0 ]
  [ "$output" = "$expected" ]
}

@test "--version prints version and exits 0" {
  expected=$(sed -n 's/^VERSION="\(.*\)"/\1/p' "$SCRIPT")
  run "$SCRIPT" --version
  [ "$status" -eq 0 ]
  [ "$output" = "$expected" ]
}

@test "unknown long flag errors and exits 1" {
  run "$SCRIPT" --bogus
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown flag --bogus"* ]]
}

@test "unknown short flag errors and exits 1" {
  run "$SCRIPT" -z
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unknown flag -z"* ]]
}

@test "extra positional argument errors and exits 1" {
  run "$SCRIPT" 1.2.3.4 5.6.7.8
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unexpected argument"* ]]
}

@test "invalid IP address errors and exits 1" {
  run "$SCRIPT" 999.1.1.1
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid IP address: 999.1.1.1"* ]]
}

@test "non-IP argument errors and exits 1" {
  run "$SCRIPT" notanip
  [ "$status" -eq 1 ]
  [[ "$output" == *"Invalid IP address: notanip"* ]]
}

@test "valid IPv6 address is accepted" {
  CURL_STUB_JSON='{"ipAddress":"2001:4860:4860::8888"}' \
    run "$SCRIPT" -j 2001:4860:4860::8888
  [ "$status" -eq 0 ]
  [ "$output" = '{"ipAddress":"2001:4860:4860::8888"}' ]
}

@test "human-readable output shows N/A for missing fields" {
  CURL_STUB_JSON='{"ipAddress":"8.8.8.8"}' run "$SCRIPT" 8.8.8.8
  [ "$status" -eq 0 ]
  [[ "$output" == *"Country: N/A (N/A)"* ]]
  [[ "$output" == *"Region: N/A"* ]]
}

@test "non-JSON API response errors cleanly" {
  CURL_STUB_JSON='<html>rate limited</html>' run "$SCRIPT" 8.8.8.8
  [ "$status" -eq 1 ]
  [[ "$output" == *"Unexpected response from API for IP 8.8.8.8"* ]]
}

@test "-i with explicit IP prints only the IP, no network call" {
  run "$SCRIPT" -i 8.8.8.8
  [ "$status" -eq 0 ]
  [ "$output" = "8.8.8.8" ]
  [ ! -e "$CURL_LOG" ]
}

@test "-ij with explicit IP prints JSON, no network call" {
  run "$SCRIPT" -ij 8.8.8.8
  [ "$status" -eq 0 ]
  [ "$output" = '{"ip":"8.8.8.8"}' ]
  [ ! -e "$CURL_LOG" ]
}

@test "combined short flags -ij4 force IPv4 for IP discovery" {
  CURL_STUB_IP="203.0.113.9" run "$SCRIPT" -ij4
  [ "$status" -eq 0 ]
  [ "$output" = '{"ip":"203.0.113.9"}' ]
  grep -q -- '-4' "$CURL_LOG"
}

@test "discovers own IP via curl when no IP given" {
  CURL_STUB_IP="198.51.100.7" run "$SCRIPT" -i
  [ "$status" -eq 0 ]
  [ "$output" = "198.51.100.7" ]
}

@test "errors when IP discovery fails" {
  CURL_STUB_EXIT=1 run "$SCRIPT" -i
  [ "$status" -eq 1 ]
  [[ "$output" == *"Could not determine public IP address"* ]]
}

@test "-j with explicit IP prints raw JSON from the API" {
  CURL_STUB_JSON='{"ipAddress":"8.8.8.8","countryName":"United States"}' \
    run "$SCRIPT" -j 8.8.8.8
  [ "$status" -eq 0 ]
  [ "$output" = '{"ipAddress":"8.8.8.8","countryName":"United States"}' ]
}

@test "errors when API lookup fails" {
  CURL_STUB_EXIT=1 run "$SCRIPT" 8.8.8.8
  [ "$status" -eq 1 ]
  [[ "$output" == *"Failed to fetch data for IP 8.8.8.8"* ]]
}

@test "human-readable output renders key-value table from API JSON" {
  CURL_STUB_JSON='{"ipAddress":"8.8.8.8","ipVersion":4,"countryName":"United States","countryCode":"US","regionName":"California","cityName":"Mountain View","zipCode":"94043","latitude":37.4,"longitude":-122.1,"timeZones":["America/Los_Angeles"]}' \
    run "$SCRIPT" 8.8.8.8
  [ "$status" -eq 0 ]
  [[ "$output" == *"IP: 8.8.8.8"* ]]
  [[ "$output" == *"Country: United States (US)"* ]]
}
