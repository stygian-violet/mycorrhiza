package shroom

import (
	"fmt"
	"slices"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/backlinks"
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
	hop := history.
		Operation(history.TypeRevertHypha).
		WithMsg(fmt.Sprintf("Revert ‘%s’ to revision %s", h.CanonicalName(), revHash)).
		WithUser(u)
	h.Lock()
	originalText, err := hyphae.FetchMycomarkupFile(h)
	if err != nil {
		h.Unlock()
		hop.Abort()
		return h, err
	}
	originalFiles := h.FilePaths()
	h.Unlock()
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
	if remove > 0 {
		hop.WithFilesRemoved(originalFiles[:remove]...)
	}
	if len(revFiles) > 0 {
		hop.WithFilesReverted(revHash, revFiles...)
	}
	revText, err := hyphae.FetchMycomarkupFile(rh)
	if err != nil {
		hop.Abort()
		return h, err
	}
	if hop.Apply().HasError() {
		return h, hop.Err()
	}
	switch rht := rh.(type) {
	case *hyphae.EmptyHypha:
		switch ht := h.(type) {
		case hyphae.ExistingHypha:
			backlinks.UpdateBacklinksAfterDelete(ht, originalText)
			categories.RemoveHyphaFromAllCategories(ht.CanonicalName())
			hyphae.DeleteHypha(ht)
		case *hyphae.EmptyHypha:
		}
	case hyphae.ExistingHypha:
		backlinks.UpdateBacklinksAfterEdit(rht, revText, originalText)
		hyphae.Insert(rht)
	}
	return rh, nil
}
