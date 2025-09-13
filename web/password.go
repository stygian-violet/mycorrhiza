package web

import (
	"fmt"
	"mime"
	"net/http"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

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
