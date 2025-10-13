package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

func handlerUserList(w http.ResponseWriter, rq *http.Request) {
	var (
		meta  = viewutil.MetaFrom(w, rq)
		canAdd = meta.U.CanProceed("admin/new-user")
		canEdit = meta.U.CanProceed("admin/users")
		canReindex = meta.U.CanProceed("admin/reindex-users")
		canManage = canAdd || canEdit || canReindex
		users []*user.User
	)

	for u := range user.YieldUsers() {
		users = append(users, u)
	}
	slices.SortFunc(users, user.Compare)

	_ = pageUserList.RenderTo(meta, map[string]any{
		"CanAdd":     canAdd,
		"CanEdit":    canEdit,
		"CanReindex": canReindex,
		"CanManage":  canManage,
		"Users":      users,
	})
}

func handlerUserSettings(w http.ResponseWriter, rq *http.Request) {
	meta := viewutil.MetaFrom(w, rq)
	_ = pageUserSettings.RenderTo(meta, map[string]any{
		"ReturnTo": cfg.Root + "hypha/" + cfg.UserHypha + "/" + meta.U.Name(),
	})
}

func handlerUserDelete(w http.ResponseWriter, rq *http.Request) {
	meta := viewutil.MetaFrom(w, rq)
	f := util.NewFormData()
	if rq.Method == "POST" {
		if err := user.DeleteUser(meta.U.Name()); err != nil {
			slog.Info("Failed to delete user", "err", err)
			f = f.WithError(err)
		} else {
			http.Redirect(w, rq, cfg.Root, http.StatusSeeOther)
			return
		}
	}
	_ = pageUserDelete.RenderTo(meta, map[string]any{
		"Form": f,
	})
}

func handlerUserChangePassword(w http.ResponseWriter, rq *http.Request) {
	meta := viewutil.MetaFrom(w, rq)
	if meta.U.IsEmpty() {
		util.HTTP404Page(w, "404 not found")
		return
	}
	f := util.FormDataFromRequest(rq, []string{"current_password", "password", "password_confirm"})

	if rq.Method == "POST" {
		currentPassword := f.Get("current_password")
		err := error(nil)

		if meta.U.IsCorrectPassword(currentPassword) {
			password := f.Get("password")
			passwordConfirm := f.Get("password_confirm")
			if password == passwordConfirm {
				var u *user.User
				u, err = meta.U.WithPassword(password)
				if err == nil {
					err = user.ReplaceUser(meta.U, u)
				}
				if err == nil {
					http.Redirect(w, rq, cfg.Root + "/settings", http.StatusSeeOther)
					return
				}
			} else {
				err = fmt.Errorf("passwords do not match")
			}
		} else {
			err = fmt.Errorf("incorrect password")
		}

		f = f.WithError(err)
		w.WriteHeader(http.StatusBadRequest)
	}

	_ = pageUserSettings.RenderTo(meta, map[string]any{
		"Form": f,
		"ReturnTo": cfg.Root + "hypha/" + cfg.UserHypha + "/" + meta.U.Name(),
	})
}
