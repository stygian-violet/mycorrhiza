package user

import (
	"fmt"
	"log/slog"
	"path"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
)

// Route — Permission level (more is more permission)
// TODO: support shell patterns?
var routePermission = map[string]int{
	"about":                  0,
	"backlinks":              0,
	"binary":                 0,
	"category":               0,
	"help":                   0,
	"history":                0,
	"hypha":                  0,
	"interwiki":              0,
	"list":                   0,
	"orphans":                0,
	"page":                   0,
	"primitive-diff":         0,
	"random":                 0,
	"recent-changes":         0,
	"recent-changes-rss":     0,
	"recent-changes-atom":    0,
	"recent-changes-json":    0,
	"rev":                    0,
	"rev-text":               0,
	"rev-binary":             0,
	"subhyphae":              0,
	"title-search":           0,
	"text":                   0,
	"text-search":            0,
	"today":                  0,
	"user-list":              0,

	"add-to-category":        1,
	"edit":                   1,
	"edit-category":          1,
	"edit-today":             1,
	"media":                  1,
	"remove-from-category":   1,
	"rename":                 1,
	"upload-binary":          1,
	"upload-text":            1,

	"remove-media":           2,

	"delete":                 3,
	"revert":                 3,
	"update-header-links":    3,

	"admin":                  4,
	"interwiki/add-entry":    4,
	"interwiki/modify-entry": 4,
	"reindex":                4,
}

func initPermissions() error {
	custom := 0
	for route, groupName := range cfg.CustomPermissions {
		err := setRoutePermission(route, groupName)
		if err != nil {
			slog.Error(
				"Failed to set route permission",
				"err", err, "route", route, "groupName", groupName,
			)
			return err
		}
		custom++
	}
	slog.Info(
		"Indexed permissions",
		"custom", custom, "total", len(routePermission),
	)
	return nil
}

func validRoute(route string) bool {
	_, ok := getRoutePermission(route)
	return ok
}

func setRoutePermission(route string, group string) error {
	route = strings.TrimPrefix(path.Clean(route), "/")
	if !validRoute(route) {
		return fmt.Errorf("invalid route '%s'", route)
	}
	g, err := GroupByName(group)
	if err != nil {
		return err
	}
	routePermission[route] = g.Permission()
	return nil
}

func getRoutePermission(route string) (int, bool) {
	for route != "." && route != "/" {
		res, ok := routePermission[route]
		// slog.Info("getRoutePermission", "route", route, "res", res, "ok", ok)
		if ok {
			return res, true
		}
		route = path.Dir(route)
	}
	return MaxPermission, false
}
