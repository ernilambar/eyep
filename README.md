# eyep

Inspect geographical and network details for an IP address or your local machine.

## Install

**Homebrew** (macOS):

```sh
brew install ernilambar/tap/eyep
```

**Standalone script** (no clone needed):

```sh
curl -L -o eyep https://github.com/ernilambar/eyep/releases/latest/download/eyep
chmod +x eyep
sudo mv eyep /usr/local/bin/
```

This is the same shell script with the release version baked in; it still needs
`curl` and `jq` installed locally.

**From source:**

```sh
git clone https://github.com/ernilambar/eyep.git
cd eyep
chmod +x eyep.sh
sudo mv eyep.sh /usr/local/bin/eyep
```

## Usage

```sh
eyep                    # Human-readable table
eyep -j                 # Raw JSON output for local machine
eyep -j 8.8.8.8         # Raw JSON output for target IP
eyep -j 8.8.8.8 | jq .  # Pretty-printed JSON
eyep -ij4               # IP-only, JSON, forced over IPv4
```

## Flags

| Flag | Long form   | Description                          |
| ---- | ----------- | ------------------------------------ |
| `-i` | `--ip-only` | Print only the IP address            |
| `-j` | `--json`    | Output raw structured JSON           |
| `-4` | `--ipv4`    | Force IPv4 network connection        |
| `-6` | `--ipv6`    | Force IPv6 network connection        |
| `-h` | `--help`    | Display help message                 |
| `-V` | `--version` | Display version information          |

Short flags may be combined, e.g. `-ij4`.

## Requirements

- `curl` — required for all network lookups
- `jq` — required only for the human-readable table; `-j` and `-i` work without it

Network calls use a 5 second connect timeout and a 10 second total timeout.
The lookup uses the free [ip-api.com](https://ip-api.com) endpoint, which is rate
limited and served over plain HTTP only.

## Testing

```sh
bats test/
```

## License

[MIT](LICENSE) © 2026 [Nilambar Sharma](https://www.nilambar.net)
