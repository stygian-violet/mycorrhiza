package shroom

import (
	"fmt"
	"slices"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/categories"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/user"
)

// Revert reverts the hypha and makes a history record about that.
func Revert(
	u *user.User,
	h hyphae.Hypha,
	revHash string,
) (hyphae.Hypha, error) {
	msg := fmt.Sprintf("Revert ‘%s’ to revision %s", h.CanonicalName(), revHash)
	hop := history.
		Operation().
		WithMsg(msg).
		WithUser(u)

	originalText, err := h.Text(hop)
	if err != nil {
		hop.Abort()
		return h, err
	}
	originalFiles := h.FilePaths()

	rh, err := hyphae.AtRevision(h.CanonicalName(), revHash)
	if err != nil {
		hop.Abort()
		return h, err
	}
	revFiles := rh.FilePaths()

	remove := 0
	for _, path := range originalFiles {
		if slices.Index(revFiles, path) < 0 {
			originalFiles[remove] = path
			remove++
		}
	}
	if remove == 0 && len(revFiles) == 0 {
		return h, nil
	}

	if remove > 0 {
		hop.WithFilesRemoved(originalFiles[:remove]...)
	}
	if len(revFiles) > 0 {
		hop.WithFilesReverted(revHash, revFiles...)
	}

	revText, err := rh.Text(hop)
	if err != nil {
		hop.Abort()
		return h, err
	}

	iop := hyphae.IndexOperation()
	rhe, rhExists := rh.(hyphae.ExistingHypha)
	he, hExists := h.(hyphae.ExistingHypha)

	switch {
	case rhExists && hExists:
		iop.WithHyphaTextChanged(he, originalText, rhe, revText)
	case rhExists:
		iop.WithHyphaCreated(rhe, revText)
	default:
		iop.WithHyphaDeleted(he, originalText)
	}

	if hop.Apply().HasError() {
		iop.Abort()
		return h, hop.Err()
	}

	if hExists && !rhExists {
		categories.RemoveHyphaeFromAllCategories(he.CanonicalName())
	}
	iop.Apply()

	return rh, nil
}
