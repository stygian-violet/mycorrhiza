package shroom

import (
	"fmt"
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/user"
)

// RemoveMedia removes media from the media hypha and makes a history record about that. If it only had media, the hypha will be deleted. If it also had text, the hypha will become textual.
func RemoveMedia(u *user.User, h *hyphae.MediaHypha) error {
	hop := history.
		Operation().
		WithFilesRemoved(h.MediaFilePath()).
		WithMsg(fmt.Sprintf("Remove media from ‘%s’", h.CanonicalName())).
		WithUser(u)

	iop := hyphae.IndexOperation()
	nh, nhExists := h.WithoutMedia().(hyphae.ExistingHypha)
	if nhExists {
		iop.WithHyphaMediaChanged(h, nh)
	} else {
		iop.WithHyphaDeleted(h, "")
	}

	if hop.Apply().HasError() {
		slog.Error("Failed to remove media", "hypha", h, "err", hop.Err())
		// FIXME: something may be wrong here
		return fmt.Errorf("Could not unattach this hypha due to internal server errors: <code>%v</code>", hop.Err())
	}

	iop.Apply()
	return nil
}
