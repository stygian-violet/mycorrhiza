package shroom

import (
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

// SetHeaderLinks initializes header links by reading the configured hypha, if there is any, or resorting to default values.
func SetHeaderLinks() error {
	var links []viewutil.HeaderLink
	userLinks, err := hyphae.
		ByName(cfg.HeaderLinksHypha).
		Text(history.FileReader())
	switch {
	case err != nil:
		slog.Error("Failed to read header links hypha", "err", err)
		fallthrough
	case userLinks == "":
		links = viewutil.DefaultHeaderLinks()
	default:
		links = hyphae.ExtractHeaderLinksFromString(
			cfg.HeaderLinksHypha, userLinks,
		)
	}
	viewutil.SetHeaderLinks(links)
	return err
}
