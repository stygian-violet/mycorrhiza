package user

import (
	"fmt"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
)

// Route — Permission level (more is more permission)
var routePermission = map[string]int{
	"text":                 0,
	"backlinks":            0,
	"history":              0,
	"text-search":          0,
	"media":                1,
	"edit":                 1,
	"upload-binary":        1,
	"rename":               1,
	"upload-text":          1,
	"add-to-category":      1,
	"remove-from-category": 1,
	"remove-media":         2,
	"update-header-links":  3,
	"delete":               3,
	"revert":               3,
	"reindex":              4,
	"admin":                4,
	"admin/shutdown":       4,
}

var groups = []string{
	"anon",
	"reader",
	"editor",
	"trusted",
	"moderator",
	"admin",
}

// Group — Permission level
var groupPermission = map[string]int{
	"anon":      0,
	"reader":    0,
	"editor":    1,
	"trusted":   2,
	"moderator": 3,
	"admin":     4,
}

func initPermissions() error {
	if err := setRoutePermission("text-search", cfg.FullTextPermission); err != nil {
		return err
	}
	return nil
}

func setRoutePermission(route string, group string) error {
	level, ok := groupPermission[group]
	if !ok {
		return fmt.Errorf("invalid group name: %s", group)
	}
	routePermission[route] = level
	return nil
}

// IsValidGroup checks whether provided user group name exists.
func IsValidGroup(group string) bool {
	for _, grp := range groups {
		if grp == group {
			return true
		}
	}
	return false
}
