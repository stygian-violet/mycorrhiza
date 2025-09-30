package shroom

import (
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
)

func Reindex() error {
	hyphaeDir := files.HyphaeDir()
	slog.Info("Reindexing hyphae", "hyphaeDir", hyphaeDir)
	return hyphae.Index(hyphaeDir)
}
