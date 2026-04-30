package cube

import (
	"iter"
	"strings"
)

func IsHostMatch(host string, whitehosts iter.Seq[string]) bool {
	if host == "" {
		return false
	}
	host = strings.ToLower(host)
	for w := range whitehosts {
		if w == "" {
			continue
		}
		rule := strings.ToLower(w)
		if host == rule {
			return true
		}
		wildcard := false
		for strings.HasPrefix(rule, "*.") {
			rule = strings.TrimPrefix(rule, "*.")
			wildcard = true
		}
		if !wildcard {
			continue
		}
		if rule == "" {
			continue
		}
		suffix := "." + rule
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
