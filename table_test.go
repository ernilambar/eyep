package main

import "testing"

func TestRenderTable(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "missing fields",
			in:   `{"status":"success","query":"8.8.8.8"}`,
			want: "IP: 8.8.8.8\nCountry: N/A (N/A)\nRegion: N/A\nCity: N/A\nCoordinates: N/A, N/A\nTimezone: N/A\nISP: N/A\nOrg: N/A\nAS: N/A",
		},
		{
			name: "full response",
			in:   `{"status":"success","query":"8.8.8.8","country":"United States","countryCode":"US","regionName":"California","city":"Mountain View","zip":"94043","lat":37.4,"lon":-122.1,"timezone":"America/Los_Angeles","isp":"Google LLC","org":"Google Public DNS","as":"AS15169 Google LLC"}`,
			want: "IP: 8.8.8.8\nCountry: United States (US)\nRegion: California\nCity: Mountain View 94043\nCoordinates: 37.4, -122.1\nTimezone: America/Los_Angeles\nISP: Google LLC\nOrg: Google Public DNS\nAS: AS15169 Google LLC",
		},
		{
			name: "city without zip",
			in:   `{"status":"success","query":"8.8.8.8","city":"Mountain View"}`,
			want: "IP: 8.8.8.8\nCountry: N/A (N/A)\nRegion: N/A\nCity: Mountain View\nCoordinates: N/A, N/A\nTimezone: N/A\nISP: N/A\nOrg: N/A\nAS: N/A",
		},
		{
			name: "only one trailing space is trimmed from city",
			in:   `{"status":"success","city":"X","zip":"  "}`,
			want: "IP: N/A\nCountry: N/A (N/A)\nRegion: N/A\nCity: X  \nCoordinates: N/A, N/A\nTimezone: N/A\nISP: N/A\nOrg: N/A\nAS: N/A",
		},
		{
			name: "empty country is kept but missing country code is not",
			in:   `{"status":"success","query":0,"lat":0,"lon":0,"country":""}`,
			want: "IP: 0\nCountry:  (N/A)\nRegion: N/A\nCity: N/A\nCoordinates: 0, 0\nTimezone: N/A\nISP: N/A\nOrg: N/A\nAS: N/A",
		},
		{
			name: "api failure message",
			in:   `{"status":"fail","message":"invalid query"}`,
			want: "Error: invalid query",
		},
		{
			name: "empty api failure message",
			in:   `{"status":"fail","message":""}`,
			want: "Error: ",
		},
		{
			name: "false api failure message falls back",
			in:   `{"status":"fail","message":false}`,
			want: "Error: Invalid IP address or API lookup error.",
		},
		{
			name: "numeric failure message",
			in:   `{"status":"fail","message":5}`,
			want: "Error: 5",
		},
		{
			name: "missing status",
			in:   `{}`,
			want: "Error: Invalid IP address or API lookup error.",
		},
		{
			name: "null response",
			in:   `null`,
			want: "Error: Invalid IP address or API lookup error.",
		},
		{
			name: "empty json stream",
			in:   "  \n",
			want: "",
		},
		{
			name: "json stream with two reports",
			in:   `{"status":"fail","message":"a"} {"status":"fail","message":"b"}`,
			want: "Error: a\nError: b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := renderTable(tt.in)
			if err != nil {
				t.Fatalf("renderTable(%q) error = %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("renderTable(%q) =\n%q\nwant\n%q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRenderTableErrors(t *testing.T) {
	inputs := []string{
		`true`,
		`5`,
		`"x"`,
		`[1]`,
		`<html>rate limited</html>`,
		`{"status":"success","city":5}`,
		`{"status":"success","zip":5}`,
	}
	for _, in := range inputs {
		if _, err := renderTable(in); err == nil {
			t.Errorf("renderTable(%q) error = nil, want error", in)
		}
	}
}
