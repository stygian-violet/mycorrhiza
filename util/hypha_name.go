package util

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"

	"git.sr.ht/~bouncepaw/mycomarkup/v5/util"
)

// BeautifulName makes the ugly name beautiful by replacing _ with spaces and using title case.
func BeautifulName(uglyName string) string {
	// Why not reuse
	return util.BeautifulName(uglyName)
}

// CanonicalName makes sure the `name` is canonical. A name is canonical if it is lowercase and all spaces are replaced with underscores.
func CanonicalName(name string) string {
	return util.CanonicalName(name)
}

// IsProfileName if the given hypha name is a profile name. It takes configuration into consideration.
//
// With default configuration, u/ is the prefix such names have. For example, u/wikimind matches. Note that u/wikimind/sub does not.
func IsProfileName(hyphaName string) bool {
	return strings.HasPrefix(hyphaName, cfg.UserHypha+"/") && strings.Count(hyphaName, "/") == 1
}

// HyphaNameFromRq extracts hypha name from http request. You have to also pass the action which is embedded in the url or several actions. For url /hypha/hypha, the action would be "hypha".
func HyphaNameFromRq(rq *http.Request, actions ...string) string {
	p := strings.TrimPrefix(rq.URL.Path, cfg.Root)
	for _, action := range actions {
		prefix := action + "/"
		if strings.HasPrefix(p, prefix) {
			return CanonicalName(p[len(prefix):])
		}
	}
	slog.Info(
		"HyphaNameFromRq: this request is invalid, fall back to home hypha",
		"path", p,
	)
	return cfg.HomeHypha
}
