package db

import "strings"

func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func EscapeHTML(s string) string {
	var result strings.Builder
	for _, c := range s {
		switch c {
		case '&':
			result.WriteString("&amp;")
		case '<':
			result.WriteString("&lt;")
		case '>':
			result.WriteString("&gt;")
		case '"':
			result.WriteString("&quot;")
		default:
			result.WriteString(string(c))
		}
	}
	return result.String()
}
