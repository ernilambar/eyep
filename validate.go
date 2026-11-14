package main

import (
	"strconv"
	"strings"
)

// validIPv4 reports whether s is a dotted-decimal IPv4 address.
//
// This is a direct port of the original shell validator rather than
// net.ParseIP so the accepted dialect stays identical: leading zeros and a
// single trailing dot are accepted, anything else is not.
func validIPv4(s string) bool {
	if s == "" {
		return false
	}
	for i := range s {
		c := s[i]
		if (c < '0' || c > '9') && c != '.' {
			return false
		}
	}
	if strings.Count(s, ".") < 3 {
		return false
	}
	// POSIX word splitting with IFS=. drops exactly one trailing delimiter.
	parts := strings.Split(s, ".")
	if parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	if len(parts) != 4 {
		return false
	}
	for _, octet := range parts {
		if octet == "" {
			return false
		}
		n, err := strconv.ParseUint(octet, 10, 64)
		if err != nil || n > 255 {
			return false
		}
	}
	return true
}

// countGroups counts colon-separated hex groups in an IPv6 fragment. It
// reports false if the fragment has an empty or over-long group.
func countGroups(s string) (int, bool) {
	if s == "" {
		return 0, true
	}
	if strings.HasPrefix(s, ":") || strings.HasSuffix(s, ":") {
		return 0, false
	}
	count := 0
	for _, group := range strings.Split(s, ":") {
		if group == "" || len(group) > 4 {
			return 0, false
		}
		for i := range group {
			if !isHex(group[i]) {
				return 0, false
			}
		}
		count++
	}
	return count, true
}

func isHex(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

// validIPv6 reports whether s is a valid IPv6 address, including forms with an
// embedded IPv4 tail such as ::ffff:192.168.0.1.
func validIPv6(s string) bool {
	// An embedded IPv4 tail stands for the last two hex groups.
	if strings.Contains(s, ".") {
		if !strings.Contains(s, ":") {
			return false
		}
		i := strings.LastIndex(s, ":")
		if !validIPv4(s[i+1:]) {
			return false
		}
		s = s[:i+1] + "0:0"
	}
	if s == "" {
		return false
	}
	for i := range s {
		if !isHex(s[i]) && s[i] != ':' {
			return false
		}
	}
	// ":::" is never valid.
	if strings.Contains(s, ":::") {
		return false
	}
	if i := strings.Index(s, "::"); i >= 0 {
		left, ok := countGroups(s[:i])
		if !ok {
			return false
		}
		right, ok := countGroups(s[i+2:])
		if !ok {
			return false
		}
		// "::" must stand in for at least one zero group.
		return left+right <= 7
	}
	n, ok := countGroups(s)
	return ok && n == 8
}

// validIP reports whether s is a valid IPv4 or IPv6 address.
func validIP(s string) bool {
	// Colons first: IPv4-mapped IPv6 addresses also contain dots.
	if strings.Contains(s, ":") {
		return validIPv6(s)
	}
	if strings.Contains(s, ".") {
		return validIPv4(s)
	}
	return false
}
