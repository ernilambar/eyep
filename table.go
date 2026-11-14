package main

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const defaultAPIError = "Invalid IP address or API lookup error."

// errUnexpectedReport marks an API response that the table renderer cannot use.
var errUnexpectedReport = errors.New("unexpected API response")

// renderTable reproduces the original jq program's output as a single string.
// It accepts a JSON stream, like jq does; an empty stream yields an empty
// string.
func renderTable(data string) (string, error) {
	dec := json.NewDecoder(strings.NewReader(data))
	dec.UseNumber()
	var reports []string
	for {
		var value any
		if err := dec.Decode(&value); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
		report, err := renderValue(value)
		if err != nil {
			return "", err
		}
		reports = append(reports, report)
	}
	return strings.Join(reports, "\n"), nil
}

func renderValue(value any) (string, error) {
	obj, ok := value.(map[string]any)
	if !ok {
		// jq treats null as an object with no fields but errors on scalars and arrays.
		if value == nil {
			return "Error: " + defaultAPIError, nil
		}
		return "", errUnexpectedReport
	}

	if status, _ := obj["status"].(string); status != "success" {
		return "Error: " + jqString(jqAlt(obj["message"], defaultAPIError)), nil
	}

	// The city/zip expression concatenates with +, which jq rejects for
	// non-string values.
	city, ok := jqAlt(obj["city"], "N/A").(string)
	if !ok {
		return "", errUnexpectedReport
	}
	if city == "" {
		city = "N/A"
	}
	zip, ok := jqAlt(obj["zip"], "").(string)
	if !ok {
		return "", errUnexpectedReport
	}
	cityLine := city + " " + zip
	// rtrimstr(" ") removes one trailing space, not all of them.
	if strings.HasSuffix(cityLine, " ") {
		cityLine = cityLine[:len(cityLine)-1]
	}
	if cityLine == "" {
		cityLine = "N/A"
	}

	rows := [][2]string{
		{"IP", jqString(jqAlt(obj["query"], "N/A"))},
		{"Country", jqString(jqAlt(obj["country"], "N/A")) + " (" + jqString(jqAlt(obj["countryCode"], "N/A")) + ")"},
		{"Region", emptyToNA(jqString(jqAlt(obj["regionName"], "N/A")))},
		{"City", cityLine},
		{"Coordinates", jqString(jqAlt(obj["lat"], "N/A")) + ", " + jqString(jqAlt(obj["lon"], "N/A"))},
		{"Timezone", emptyToNA(jqString(jqAlt(obj["timezone"], "N/A")))},
		{"ISP", emptyToNA(jqString(jqAlt(obj["isp"], "N/A")))},
		{"Org", emptyToNA(jqString(jqAlt(obj["org"], "N/A")))},
		{"AS", emptyToNA(jqString(jqAlt(obj["as"], "N/A")))},
	}

	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(row[0])
		b.WriteString(": ")
		b.WriteString(row[1])
	}
	return b.String(), nil
}

// jqAlt is jq's // operator: a null or false value falls back to fallback.
func jqAlt(value, fallback any) any {
	if value == nil {
		return fallback
	}
	if b, ok := value.(bool); ok && !b {
		return fallback
	}
	return value
}

// jqString renders a value the way jq string interpolation does.
func jqString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case json.Number:
		return v.String()
	case bool:
		if v {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		encoded, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}

func emptyToNA(s string) string {
	if s == "" {
		return "N/A"
	}
	return s
}
