package name

import (
	"strings"
)

// Clean cleans the name of
func Clean(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}
