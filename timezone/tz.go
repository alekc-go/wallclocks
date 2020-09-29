package timezone

import (
	"strings"
)

// CleanName cleans the name of
func CleanName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}
