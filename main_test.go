package main

import (
	"bytes"
	"errors"
	"regexp"
	"testing"
)

const wantUsage = `Usage: eyep [FLAGS] [IP_ADDRESS]

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
`

type stubBackend struct {
	discoverBody string
	discoverErr  error
	lookupBody   string
	lookupErr    error

	discoverCalls int
	lookupCalls   int
	lookupIP      string
}

func (s *stubBackend) discover() (string, error) {
	s.discoverCalls++
	return s.discoverBody, s.discoverErr
}

func (s *stubBackend) lookup(ip string) (string, error) {
	s.lookupCalls++
	s.lookupIP = ip
	return s.lookupBody, s.lookupErr
}

type runResult struct {
	stdout string
	stderr string
	code   int
	family string
}

func runStubWith(t *testing.T, stub *stubBackend, args ...string) runResult {
	t.Helper()
	var stdout, stderr bytes.Buffer
	family := ""
	code := runWith(append([]string{"eyep"}, args...), &stdout, &stderr, func(f string) backend {
		family = f
		return stub
	})
	return runResult{stdout: stdout.String(), stderr: stderr.String(), code: code, family: family}
}

func runStub(t *testing.T, args ...string) runResult {
	t.Helper()
	return runStubWith(t, &stubBackend{}, args...)
}

func TestHelp(t *testing.T) {
	for _, flag := range []string{"-h", "--help"} {
		r := runStub(t, flag)
		if r.code != 0 {
			t.Errorf("%s exit = %d, want 0", flag, r.code)
		}
		if r.stderr != "" {
			t.Errorf("%s stderr = %q, want empty", flag, r.stderr)
		}
		if r.stdout != wantUsage {
			t.Errorf("%s stdout =\n%q\nwant\n%q", flag, r.stdout, wantUsage)
		}
	}
}

func TestUsageUsesProgramBasename(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runWith([]string{"/usr/local/bin/eyep", "-h"}, &stdout, &stderr, func(string) backend {
		return &stubBackend{}
	})
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if want := "Usage: eyep [FLAGS] [IP_ADDRESS]\n"; !bytes.HasPrefix(stdout.Bytes(), []byte(want)) {
		t.Errorf("stdout prefix = %q, want %q", stdout.String(), want)
	}
}

func TestHelpInsideCluster(t *testing.T) {
	r := runStub(t, "-ih")
	if r.code != 0 || r.stderr != "" || r.stdout != wantUsage {
		t.Errorf("-ih = %+v, want help on stdout", r)
	}
}

func TestHelpStopsParsing(t *testing.T) {
	r := runStub(t, "-h", "--bogus")
	if r.code != 0 || r.stdout != wantUsage {
		t.Errorf("-h --bogus = %+v, want help", r)
	}
}

func TestVersion(t *testing.T) {
	if !regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(version) {
		t.Fatalf("version %q is not semver", version)
	}
	for _, flag := range []string{"-V", "--version"} {
		r := runStub(t, flag)
		if r.code != 0 || r.stderr != "" || r.stdout != version+"\n" {
			t.Errorf("%s = %+v, want %q on stdout", flag, r, version+"\n")
		}
	}
}

func TestVersionAfterPositional(t *testing.T) {
	r := runStub(t, "8.8.8.8", "-V")
	if r.code != 0 || r.stdout != version+"\n" {
		t.Errorf("8.8.8.8 -V = %+v, want version", r)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"--bogus"}, "Error: Unknown flag --bogus\n\n"},
		{[]string{"-z"}, "Error: Unknown flag -z\n\n"},
		{[]string{"-iz"}, "Error: Unknown flag -z\n\n"},
		{[]string{"--ip-only=1"}, "Error: Unknown flag --ip-only=1\n\n"},
		{[]string{"1.2.3.4", "5.6.7.8"}, "Error: Unexpected argument 5.6.7.8\n\n"},
	}

	for _, tt := range tests {
		stub := &stubBackend{}
		r := runStubWith(t, stub, tt.args...)
		if r.code != 1 {
			t.Errorf("%v exit = %d, want 1", tt.args, r.code)
		}
		if r.stdout != "" {
			t.Errorf("%v stdout = %q, want empty", tt.args, r.stdout)
		}
		if want := tt.want + wantUsage; r.stderr != want {
			t.Errorf("%v stderr =\n%q\nwant\n%q", tt.args, r.stderr, want)
		}
		if stub.discoverCalls != 0 || stub.lookupCalls != 0 {
			t.Errorf("%v made network calls: %+v", tt.args, stub)
		}
	}
}

func TestInvalidIP(t *testing.T) {
	for _, ip := range []string{"999.1.1.1", "notanip", "::ffff:999.1.1.1"} {
		stub := &stubBackend{}
		r := runStubWith(t, stub, "-i", ip)
		if r.code != 1 {
			t.Errorf("ip %q exit = %d, want 1", ip, r.code)
		}
		if r.stdout != "" {
			t.Errorf("ip %q stdout = %q, want empty", ip, r.stdout)
		}
		if want := "Error: Invalid IP address: " + ip + "\n"; r.stderr != want {
			t.Errorf("ip %q stderr = %q, want %q", ip, r.stderr, want)
		}
		if stub.discoverCalls != 0 || stub.lookupCalls != 0 {
			t.Errorf("ip %q made network calls", ip)
		}
	}
}

func TestExplicitIPOnlyMakesNoNetworkCall(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{[]string{"-i", "8.8.8.8"}, "8.8.8.8\n"},
		{[]string{"-i", "1.2.3.4."}, "1.2.3.4.\n"},
		{[]string{"-ij", "8.8.8.8"}, "{\"ip\":\"8.8.8.8\"}\n"},
		{[]string{"-ij", "0.0.0.0"}, "{\"ip\":\"0.0.0.0\"}\n"},
		{[]string{"-ij", "255.255.255.255"}, "{\"ip\":\"255.255.255.255\"}\n"},
		{[]string{"-ij", "2001:4860:4860::8888"}, "{\"ip\":\"2001:4860:4860::8888\"}\n"},
		{[]string{"-ij", "::1"}, "{\"ip\":\"::1\"}\n"},
		{[]string{"-ij", "::ffff:192.168.1.1"}, "{\"ip\":\"::ffff:192.168.1.1\"}\n"},
		{[]string{"-ij", "1:2:3:4:5:6:1.2.3.4"}, "{\"ip\":\"1:2:3:4:5:6:1.2.3.4\"}\n"},
		{[]string{"-ij", "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"}, "{\"ip\":\"2001:db8:ffff:ffff:ffff:ffff:ffff:ffff\"}\n"},
	}

	for _, tt := range tests {
		stub := &stubBackend{}
		r := runStubWith(t, stub, tt.args...)
		if r.code != 0 {
			t.Errorf("%v exit = %d, want 0", tt.args, r.code)
		}
		if r.stdout != tt.want {
			t.Errorf("%v stdout = %q, want %q", tt.args, r.stdout, tt.want)
		}
		if stub.discoverCalls != 0 || stub.lookupCalls != 0 {
			t.Errorf("%v made network calls", tt.args)
		}
	}
}

func TestArgumentSeparator(t *testing.T) {
	r := runStub(t, "-ij", "--", "8.8.8.8")
	if r.code != 0 || r.stdout != "{\"ip\":\"8.8.8.8\"}\n" {
		t.Errorf("-ij -- 8.8.8.8 = %+v", r)
	}

	r = runStub(t, "-i", "--", "-h")
	if r.code != 1 || r.stderr != "Error: Invalid IP address: -h\n" {
		t.Errorf("-i -- -h = %+v", r)
	}

	r = runStub(t, "-i", "1.2.3.4", "--", "5.6.7.8")
	if r.code != 1 || r.stderr != "Error: Unexpected argument 5.6.7.8\n\n"+wantUsage {
		t.Errorf("-i 1.2.3.4 -- 5.6.7.8 = %+v", r)
	}

	r = runStub(t, "--", "--help")
	if r.code != 1 || r.stderr != "Error: Invalid IP address: --help\n" {
		t.Errorf("-- --help = %+v", r)
	}

	r = runStub(t, "-")
	if r.code != 1 || r.stderr != "Error: Invalid IP address: -\n" {
		t.Errorf("- = %+v", r)
	}
}

func TestJSONOutputForExplicitIP(t *testing.T) {
	stub := &stubBackend{
		lookupBody: `{"ipAddress":"8.8.8.8","countryName":"United States"}`,
	}
	r := runStubWith(t, stub, "-j", "8.8.8.8")
	if r.code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr %q)", r.code, r.stderr)
	}
	if want := stub.lookupBody + "\n"; r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
	if stub.discoverCalls != 0 || stub.lookupCalls != 1 || stub.lookupIP != "8.8.8.8" {
		t.Errorf("unexpected calls: %+v", stub)
	}
}

func TestJSONOutputForDiscoveredIP(t *testing.T) {
	stub := &stubBackend{
		discoverBody: "198.51.100.7\n",
		lookupBody:   `{"status":"success","query":"198.51.100.7"}`,
	}
	r := runStubWith(t, stub, "-j")
	if r.code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr %q)", r.code, r.stderr)
	}
	if want := stub.lookupBody + "\n"; r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
	if stub.discoverCalls != 1 || stub.lookupCalls != 1 || stub.lookupIP != "198.51.100.7" {
		t.Errorf("unexpected calls: %+v", stub)
	}
	if r.family != "tcp4" {
		t.Errorf("family = %q, want tcp4", r.family)
	}
}

func TestJSONTrailingNewlinesAreTrimmed(t *testing.T) {
	stub := &stubBackend{lookupBody: "{\"a\":1}\n\n"}
	r := runStubWith(t, stub, "-j", "8.8.8.8")
	if r.code != 0 || r.stdout != "{\"a\":1}\n" {
		t.Errorf("-j = %+v", r)
	}
}

func TestIPOnlyDiscoverySkipsLookup(t *testing.T) {
	stub := &stubBackend{discoverBody: "203.0.113.9"}
	r := runStubWith(t, stub, "-ij")
	if r.code != 0 || r.stdout != "{\"ip\":\"203.0.113.9\"}\n" {
		t.Errorf("-ij = %+v", r)
	}
	if stub.discoverCalls != 1 || stub.lookupCalls != 0 {
		t.Errorf("unexpected calls: %+v", stub)
	}
}

func TestDiscoveryFailure(t *testing.T) {
	tests := []struct {
		name string
		stub *stubBackend
	}{
		{"request error", &stubBackend{discoverErr: errors.New("down")}},
		{"empty response", &stubBackend{discoverBody: ""}},
		{"html response", &stubBackend{discoverBody: "<html>rate limited</html>"}},
		{"invalid ip", &stubBackend{discoverBody: "not-an-ip"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := runStubWith(t, tt.stub, "-ij")
			if r.code != 1 {
				t.Errorf("exit = %d, want 1", r.code)
			}
			if r.stdout != "" {
				t.Errorf("stdout = %q, want empty", r.stdout)
			}
			if want := "Error: Could not determine public IP address.\n"; r.stderr != want {
				t.Errorf("stderr = %q, want %q", r.stderr, want)
			}
			if tt.stub.lookupCalls != 0 {
				t.Errorf("lookup was called for invalid discovery")
			}
		})
	}
}

func TestLookupFailure(t *testing.T) {
	tests := []struct {
		name string
		stub *stubBackend
		args []string
		ip   string
	}{
		{
			name: "discovered ip",
			stub: &stubBackend{discoverBody: "203.0.113.9", lookupErr: errors.New("down")},
			args: []string{"-j"},
			ip:   "203.0.113.9",
		},
		{
			name: "explicit ip",
			stub: &stubBackend{lookupErr: errors.New("down")},
			args: []string{"8.8.8.8"},
			ip:   "8.8.8.8",
		},
		{
			name: "empty response",
			stub: &stubBackend{lookupBody: ""},
			args: []string{"8.8.8.8"},
			ip:   "8.8.8.8",
		},
		{
			name: "newline only response",
			stub: &stubBackend{lookupBody: "\n\n"},
			args: []string{"8.8.8.8"},
			ip:   "8.8.8.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := runStubWith(t, tt.stub, tt.args...)
			if r.code != 1 {
				t.Errorf("exit = %d, want 1", r.code)
			}
			if r.stdout != "" {
				t.Errorf("stdout = %q, want empty", r.stdout)
			}
			if want := "Error: Failed to fetch data for IP " + tt.ip + ".\n"; r.stderr != want {
				t.Errorf("stderr = %q, want %q", r.stderr, want)
			}
		})
	}
}

func TestTableOutput(t *testing.T) {
	stub := &stubBackend{
		lookupBody: `{"status":"success","query":"8.8.8.8"}`,
	}
	r := runStubWith(t, stub, "8.8.8.8")
	if r.code != 0 {
		t.Fatalf("exit = %d, want 0 (stderr %q)", r.code, r.stderr)
	}
	want := "IP: 8.8.8.8\nCountry: N/A (N/A)\nRegion: N/A\nCity: N/A\nCoordinates: N/A, N/A\nTimezone: N/A\nISP: N/A\nOrg: N/A\nAS: N/A\n"
	if r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
}

func TestTableAPIFailure(t *testing.T) {
	stub := &stubBackend{lookupBody: `{"status":"fail","message":"invalid query"}`}
	r := runStubWith(t, stub, "8.8.8.8")
	if r.code != 1 {
		t.Errorf("exit = %d, want 1", r.code)
	}
	if r.stdout != "" {
		t.Errorf("stdout = %q, want empty", r.stdout)
	}
	if r.stderr != "Error: invalid query\n" {
		t.Errorf("stderr = %q", r.stderr)
	}
}

func TestTableUnexpectedResponse(t *testing.T) {
	stub := &stubBackend{lookupBody: "<html>rate limited</html>"}
	r := runStubWith(t, stub, "8.8.8.8")
	if r.code != 1 {
		t.Errorf("exit = %d, want 1", r.code)
	}
	if r.stdout != "" {
		t.Errorf("stdout = %q, want empty", r.stdout)
	}
	if want := "Error: Unexpected response from API for IP 8.8.8.8.\n"; r.stderr != want {
		t.Errorf("stderr = %q, want %q", r.stderr, want)
	}
}

func TestJSONOutputIsNotValidated(t *testing.T) {
	stub := &stubBackend{lookupBody: "<html>rate limited</html>"}
	r := runStubWith(t, stub, "-j", "8.8.8.8")
	if r.code != 0 || r.stdout != "<html>rate limited</html>\n" {
		t.Errorf("-j = %+v", r)
	}
}

func TestAddressFamily(t *testing.T) {
	tests := []struct {
		args   []string
		family string
	}{
		{[]string{"-i"}, "tcp4"},
		{[]string{"-ij4"}, "tcp4"},
		{[]string{"-ij6"}, "tcp6"},
		{[]string{"--ipv4", "-i"}, "tcp4"},
		{[]string{"--ipv6", "-i"}, "tcp6"},
		{[]string{"-6", "-i"}, "tcp6"},
	}

	for _, tt := range tests {
		stub := &stubBackend{discoverBody: "203.0.113.9"}
		r := runStubWith(t, stub, tt.args...)
		if r.code != 0 {
			t.Errorf("%v exit = %d, want 0 (stderr %q)", tt.args, r.code, r.stderr)
		}
		if r.family != tt.family {
			t.Errorf("%v family = %q, want %q", tt.args, r.family, tt.family)
		}
	}
}
