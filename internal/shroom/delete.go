package shroom

import (
	"errors"
	"fmt"
	"iter"
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/categories"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/user"
)

var ErrDeleteEmpty = errors.New("nothing to delete")

// Delete deletes the hypha and makes a history record about that.
func Delete(u *user.User, h hyphae.Hypha, recursive bool) error {
	hop := history.
		Operation().
		WithUser(u)
	iop := hyphae.IndexOperation()

	names := []string(nil)
	files := []string(nil)
	for hypha := range yieldHyphaeToDelete(h, recursive, iop) {
		text, err := hypha.Text(hop)
		if err != nil {
			slog.Error("Failed to read hypha text", "hypha", hypha, "err", err)
			hop.Abort()
			iop.Abort()
			return err
		}
		names = append(names, hypha.CanonicalName())
		files = append(files, hypha.FilePaths()...)
		iop.WithHyphaDeleted(hypha, text)
	}
	if names == nil {
		iop.Abort()
		hop.Abort()
		return ErrDeleteEmpty
	}

	var msg string
	if len(names) > 1 || names[0] != h.CanonicalName() {
		msg = "Delete ‘%s’ recursively"
	} else {
		msg = "Delete ‘%s’"
	}

	hop.
		WithMsg(fmt.Sprintf(msg, h.CanonicalName())).
		WithFilesRemoved(files...).
		Apply()
	if hop.HasError() {
		iop.Abort()
		return hop.Err()
	}

	categories.RemoveHyphaeFromAllCategories(names...)
	iop.Apply()
	return nil
}

func yieldHyphaeToDelete(
	h hyphae.Hypha,
	recursive bool,
	iop *hyphae.Op,
) iter.Seq[hyphae.ExistingHypha] {
	return func(yield func(hyphae.ExistingHypha) bool) {
		if he, ok := h.(hyphae.ExistingHypha); ok && !yield(he) || !recursive {
			return
		}
		for subh := range iop.YieldSubhyphae(h) {
			if !yield(subh) {
				return
			}
		}
	}
}
