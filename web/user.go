package web

import (
	"fmt"
	"mime"
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

func handlerUserChangePassword(w http.ResponseWriter, rq *http.Request) {
	u := user.FromRequest(rq)
	if u.IsEmpty() {
		util.HTTP404Page(w, "404 not found")
		return
	}

	f := util.FormDataFromRequest(rq, []string{"current_password", "password", "password_confirm"})

	if rq.Method == "POST" {
		currentPassword := f.Get("current_password")

		if u.IsCorrectPassword(currentPassword) {
			password := f.Get("password")
			passwordConfirm := f.Get("password_confirm")
			if password == passwordConfirm {
				nu, err := u.WithPassword(password)
				if err != nil {
					f = f.WithError(err)
				} else if err = user.ReplaceUser(u, nu); err != nil {
					f = f.WithError(err)
				} else {
					http.Redirect(w, rq, cfg.Root, http.StatusSeeOther)
					return
				}
			} else {
				err := fmt.Errorf("passwords do not match")
				f = f.WithError(err)
			}
		} else {
			err := fmt.Errorf("incorrect password")
			f = f.WithError(err)
		}
	}

	if f.HasError() {
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(".html"))

	_ = pageChangePassword.RenderTo(
		viewutil.MetaFrom(w, rq),
		map[string]any{
			"Form": f,
			"U":    u,
		},
	)
}
