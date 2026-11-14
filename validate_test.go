package main

import "testing"

func TestValidIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		// IPv4
		{"0.0.0.0", true},
		{"255.255.255.255", true},
		{"8.8.8.8", true},
		{"01.02.03.04", true},
		{"08.1.1.1", true},
		{"0000000000000000000001.2.3.4", true},
		{"1.2.3.4.", true}, // the shell validator drops one trailing delimiter
		{"999.1.1.1", false},
		{"256.0.0.1", false},
		{"0256.1.1.1", false},
		{"1.2.3", false},
		{"1.2.3.4.5", false},
		{"1.2.3.x", false},
		{"notanip", false},
		{"1.2.3.4..", false},
		{".1.2.3", false},
		{"1.2.3..4", false},
		{"...1.2", false},
		{"1.2.3.4 ", false},
		{"", false},

		// IPv6
		{"::", true},
		{"::1", true},
		{"1::", true},
		{"1::2", true},
		{"2001:db8::1", true},
		{"fe80::1", true},
		{"2001:4860:4860::8888", true},
		{"0001:2:3:4:5:6:7:8", true},
		{"2001:db8:ffff:ffff:ffff:ffff:ffff:ffff", true},
		{"::ffff:192.168.1.1", true},
		{"1:2:3:4:5:6:1.2.3.4", true},
		{"::1.2.3.4", true},
		{"::ffff:192.168.1.1.", true},
		{":", false},
		{":::", false},
		{"::::", false},
		{"1::2::3", false},
		{":1:2", false},
		{"1:2:", false},
		{"gggg::1", false},
		{"1:2:3:4:5:6:7:8:9:10", false},
		{"12345::1", false},
		{"fffff::1", false},
		{"00000000000000001::1", false},
		{"1:2:3:4:5:6:7:8::9", false},
		{"::1:2:3:4:5:6:7:8", false},
		{"1:2:3:4:5:6:7:1.2.3.4", false},
		{"::ffff:999.1.1.1", false},
		{"::1.2.3", false},
		{"::ffff:1.2.3.4.5", false},
		{"1:2:3:4:5:6:7:8:", false},
		{":1:2:3:4:5:6:7:8", false},
		{"1.2.3.4::", false},
	}

	for _, tt := range tests {
		if got := validIP(tt.ip); got != tt.want {
			t.Errorf("validIP(%q) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}
