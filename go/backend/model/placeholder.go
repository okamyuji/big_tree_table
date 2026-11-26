package model

import (
	"strconv"
	"strings"
)

// DollarPlaceholders rewrites the `?` placeholders that BuildQuery and
// BuildCountQuery emit into PostgreSQL's $1, $2, …. Those builders never put a
// literal `?` in the SQL text (user values travel as args), so every `?` is a
// placeholder.
func DollarPlaceholders(query string) string {
	var b strings.Builder
	b.Grow(len(query) + 8)
	n := 0
	for i := 0; i < len(query); i++ {
		if query[i] != '?' {
			b.WriteByte(query[i])
			continue
		}
		n++
		b.WriteByte('$')
		b.WriteString(strconv.Itoa(n))
	}
	return b.String()
}
