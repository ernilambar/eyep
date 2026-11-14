package main

import (
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed VERSION
var versionFile string

// version comes from the VERSION file, the single source of truth for releases.
var version = strings.TrimSpace(versionFile)

func main() {
	os.Exit(run(os.Args, os.Stdout, os.Stderr))
}

// usage writes the help message to w.
func usage(w io.Writer, prog string) {
	fmt.Fprintf(w, `Usage: %s [FLAGS] [IP_ADDRESS]

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
`, prog)
}

type options struct {
	ipOnly  bool
	jsonOut bool
	family  string
	target  string
}

type action int

const (
	actionRun action = iota
	actionHelp
	actionVersion
	actionError
)

type parseResult struct {
	opts options
	act  action
	msg  string
}

// parseArgs mirrors the original POSIX flag parser, including short-flag
// clusters, immediate help/version handling and "--" termination.
func parseArgs(args []string) parseResult {
	opts := options{family: "tcp4"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			i++
			if i < len(args) {
				if opts.target == "" {
					opts.target = args[i]
					i++
				}
				if i < len(args) {
					return parseResult{opts: opts, act: actionError, msg: "Unexpected argument " + args[i]}
				}
			}
			return parseResult{opts: opts, act: actionRun}
		case arg == "--ip-only":
			opts.ipOnly = true
		case arg == "--json":
			opts.jsonOut = true
		case arg == "--ipv4":
			opts.family = "tcp4"
		case arg == "--ipv6":
			opts.family = "tcp6"
		case arg == "--help":
			return parseResult{opts: opts, act: actionHelp}
		case arg == "--version":
			return parseResult{opts: opts, act: actionVersion}
		case strings.HasPrefix(arg, "--"):
			return parseResult{opts: opts, act: actionError, msg: "Unknown flag " + arg}
		case len(arg) > 1 && arg[0] == '-':
			for _, flag := range arg[1:] {
				switch flag {
				case 'i':
					opts.ipOnly = true
				case 'j':
					opts.jsonOut = true
				case '4':
					opts.family = "tcp4"
				case '6':
					opts.family = "tcp6"
				case 'h':
					return parseResult{opts: opts, act: actionHelp}
				case 'V':
					return parseResult{opts: opts, act: actionVersion}
				default:
					return parseResult{opts: opts, act: actionError, msg: fmt.Sprintf("Unknown flag -%c", flag)}
				}
			}
		default:
			if opts.target != "" {
				return parseResult{opts: opts, act: actionError, msg: "Unexpected argument " + arg}
			}
			opts.target = arg
		}
	}
	return parseResult{opts: opts, act: actionRun}
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWith(args, stdout, stderr, func(family string) backend {
		return newHTTPBackend(family)
	})
}

// runWith is run with an injectable backend factory, used by tests.
func runWith(args []string, stdout, stderr io.Writer, newBackend func(family string) backend) int {
	prog := "eyep"
	if len(args) > 0 {
		prog = filepath.Base(args[0])
	}
	var argv []string
	if len(args) > 1 {
		argv = args[1:]
	}

	res := parseArgs(argv)
	switch res.act {
	case actionHelp:
		usage(stdout, prog)
		return 0
	case actionVersion:
		fmt.Fprintln(stdout, version)
		return 0
	case actionError:
		fmt.Fprintf(stderr, "Error: %s\n\n", res.msg)
		usage(stderr, prog)
		return 1
	}

	opts := res.opts
	if opts.target != "" && !validIP(opts.target) {
		fmt.Fprintf(stderr, "Error: Invalid IP address: %s\n", opts.target)
		return 1
	}

	api := newBackend(opts.family)
	ip := opts.target
	if ip == "" {
		body, err := api.discover()
		ip = strings.TrimRight(body, "\n")
		if err != nil || ip == "" || !validIP(ip) {
			fmt.Fprintln(stderr, "Error: Could not determine public IP address.")
			return 1
		}
	}

	if opts.ipOnly {
		if opts.jsonOut {
			fmt.Fprintf(stdout, "{\"ip\":\"%s\"}\n", ip)
		} else {
			fmt.Fprintln(stdout, ip)
		}
		return 0
	}

	body, err := api.lookup(ip)
	data := strings.TrimRight(body, "\n")
	if err != nil || data == "" {
		fmt.Fprintf(stderr, "Error: Failed to fetch data for IP %s.\n", ip)
		return 1
	}

	if opts.jsonOut {
		fmt.Fprintln(stdout, data)
		return 0
	}

	table, err := renderTable(data)
	if err != nil {
		fmt.Fprintf(stderr, "Error: Unexpected response from API for IP %s.\n", ip)
		return 1
	}
	if strings.HasPrefix(table, "Error:") {
		fmt.Fprintln(stderr, table)
		return 1
	}
	fmt.Fprintln(stdout, table)
	return 0
}
