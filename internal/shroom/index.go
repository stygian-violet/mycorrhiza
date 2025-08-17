package shroom

import (
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/internal/backlinks"
	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
)

func Reindex() {
	slog.Info("Reindexing hyphae", "hyphaeDir", files.HyphaeDir())
	hyphae.Index(files.HyphaeDir())
	backlinks.IndexBacklinks()
}
