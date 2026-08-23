package aggregate

import (
	"regexp"
	"strings"
)

// Session names are derived as org:repo:bookmark. The gateway is the ONLY
// place names are assembled, so it must enforce the component contract here
// (mirroring repo-extension's naming.go) — an invalid component would create
// an agent session whose lifecycle event is permanently discarded downstream
// (no workspace, ever). Fail fast at the entry point instead.
//
// Component rules:
//   - ^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$ (charset implies no ':')
//   - no ".." (path traversal / jj rule)
//   - no trailing "." / ".lock" (jj/git ref rules)

var componentRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

// validComponent reports whether s is acceptable as an org/repo/bookmark name.
func validComponent(s string) bool {
	if !componentRe.MatchString(s) {
		return false
	}
	if strings.Contains(s, ":") {
		return false // reserved separator (defense in depth)
	}
	if strings.Contains(s, "..") {
		return false
	}
	if strings.HasSuffix(s, ".") || strings.HasSuffix(s, ".lock") {
		return false
	}
	return true
}

// requireComponents validates all components or returns an error naming the
// first offender.
func requireComponents(pairs ...[2]string) (string, bool) {
	for _, p := range pairs {
		if !validComponent(p[1]) {
			return p[0], false
		}
	}
	return "", true
}
