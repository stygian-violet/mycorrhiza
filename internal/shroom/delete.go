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
func Delete(u *user.User, h hyphae.ExistingHypha) error {
	hop := history.
		Operation().
		WithMsg(fmt.Sprintf("Delete ‘%s’", h.CanonicalName())).
		WithUser(u)
	originalText, err := h.Text(hop)
	if err != nil {
		slog.Error("Failed to read hypha text", "hypha", h, "err", err)
		hop.Abort()
		return err
	}
	iop := hyphae.IndexOperation()
	hop.WithFilesRemoved(h.FilePaths()...)
	iop.WithHyphaDeleted(h, originalText)
	hop.Apply()
	if hop.HasError() {
		iop.Abort()
		return hop.Err()
	}
	categories.RemoveHyphaFromAllCategories(h.CanonicalName())
	iop.Apply()
	return nil
}
