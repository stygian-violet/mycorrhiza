package shroom

import (
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/mycoopts"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"

	"git.sr.ht/~bouncepaw/mycomarkup/v5"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/blocks"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/mycocontext"
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
		links = defaultHeaderLinks()
	default:
		links = parseHeaderLinks(userLinks)
	}
	viewutil.SetHeaderLinks(links)
	return err
}

// defaultHeaderLinks returns the default list of: home hypha, recent changes, hyphae list, random hypha.
func defaultHeaderLinks() []viewutil.HeaderLink {
	return []viewutil.HeaderLink{
		{cfg.Root+"recent-changes", "Recent changes"},
		{cfg.Root+"list", "All hyphae"},
		{cfg.Root+"random", "Random"},
		{cfg.Root+"help", "Help"},
		{cfg.Root+"category", "Categories"},
	}
}

// parseHeaderLinks extracts all rocketlinks from the given text and returns them as header links.
func parseHeaderLinks(text string) []viewutil.HeaderLink {
	headerLinks := []viewutil.HeaderLink{}
	ctx, _ := mycocontext.ContextFromStringInput(text, mycoopts.MarkupOptions(""))
	// We call for side-effects
	_ = mycomarkup.BlockTree(ctx, func(block blocks.Block) {
		switch launchpad := block.(type) {
		case blocks.LaunchPad:
			for _, rocket := range launchpad.Rockets {
				headerLinks = append(headerLinks, viewutil.HeaderLink{
					Href:    rocket.LinkHref(ctx),
					Display: rocket.DisplayedText(),
				})
			}
		}
	})
	return headerLinks
}
