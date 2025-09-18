package shroom

import (
	"fmt"
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/categories"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/user"
)

// Delete deletes the hypha and makes a history record about that.
func Delete(u *user.User, h hyphae.ExistingHypha, recursive bool) error {
	hop := history.
		Operation().
		WithUser(u)
	iop := hyphae.IndexOperation()
	hyphae := findHyphaeToDelete(iop, h, recursive)
	var msg string
	if len(hyphae) > 1 {
		msg = "Delete ‘%s’ recursively"
	} else {
		msg = "Delete ‘%s’"
	}
	files := []string(nil)
	for _, hypha := range hyphae {
		text, err := hypha.Text(hop)
		if err != nil {
			slog.Error("Failed to read hypha text", "hypha", hypha, "err", err)
			hop.Abort()
			iop.Abort()
			return err
		}
		files = append(files, hypha.FilePaths()...)
		iop.WithHyphaDeleted(hypha, text)
	}
	hop.
		WithMsg(fmt.Sprintf(msg, h.CanonicalName())).
		WithFilesRemoved(files...).
		Apply()
	if hop.HasError() {
		iop.Abort()
		return hop.Err()
	}
	for _, h := range hyphae {
		categories.RemoveHyphaFromAllCategories(h.CanonicalName())
	}
	iop.Apply()
	return nil
}

func findHyphaeToDelete(
	iop *hyphae.Op,
	h hyphae.ExistingHypha,
	recursive bool,
) []hyphae.ExistingHypha {
	res := []hyphae.ExistingHypha{h}
	if recursive {
		for subh := range iop.YieldSubhyphae(h) {
			res = append(res, subh)
		}
	}
	return res
}
