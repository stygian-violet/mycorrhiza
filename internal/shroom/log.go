package shroom

import (
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/user"
)

func rejectRenameLog(h hyphae.Hypha, u *user.User, errmsg string) {
	u.RLock()
	slog.Info("Reject rename",
		"hyphaName", h.CanonicalName(),
		"username", u.Name,
		"errmsg", errmsg)
	u.RUnlock()
}

func rejectRemoveMediaLog(h hyphae.Hypha, u *user.User, errmsg string) {
	u.RLock()
	slog.Info("Reject remove media",
		"hyphaName", h.CanonicalName(),
		"username", u.Name,
		"errmsg", errmsg)
	u.RUnlock()
}

func rejectEditLog(h hyphae.Hypha, u *user.User, errmsg string) {
	u.RLock()
	slog.Info("Reject edit",
		"hyphaName", h.CanonicalName(),
		"username", u.Name,
		"errmsg", errmsg)
	u.RUnlock()
}

func rejectUploadMediaLog(h hyphae.Hypha, u *user.User, errmsg string) {
	u.RLock()
	slog.Info("Reject upload media",
		"hyphaName", h.CanonicalName(),
		"username", u.Name,
		"errmsg", errmsg)
	u.RUnlock()
}
