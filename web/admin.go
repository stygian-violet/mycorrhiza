package web

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/process"
	"github.com/bouncepaw/mycorrhiza/internal/shroom"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
	"github.com/gorilla/mux"
)

const adminTranslationRu = `
{{define "panel title"}}Панель админстратора{{end}}
{{define "panel safe section title"}}Безопасная секция{{end}}
{{define "panel link about"}}Об этой вики{{end}}
{{define "panel update header"}}Обновить ссылки в верхней панели{{end}}
{{define "panel link user list"}}Список пользователей{{end}}
{{define "panel users"}}Управление пользователями{{end}}
{{define "panel unsafe section title"}}Опасная секция{{end}}
{{define "panel shutdown"}}Выключить вики{{end}}
{{define "panel reindex hyphae"}}Переиндексировать гифы{{end}}
{{define "panel interwiki"}}Интервики{{end}}

{{define "manage users"}}Управление пользователями{{end}}
{{define "create user"}}Создать пользователя{{end}}
{{define "reindex users"}}Переиндексировать пользователей{{end}}
{{define "name"}}Имя{{end}}
{{define "group"}}Группа{{end}}
{{define "registered at"}}Зарегистрирован{{end}}
{{define "actions"}}Действия{{end}}
{{define "edit"}}Изменить{{end}}

{{define "new user"}}Новый пользователь{{end}}
{{define "password"}}Пароль{{end}}
{{define "confirm password"}}Подтвердить пароль{{end}}
{{define "change password"}}Изменить пароль{{end}}
{{define "non local password change"}}Поменять пароль можно только у локальных пользователей.{{end}}
{{define "create"}}Создать{{end}}

{{define "change group"}}Изменить группу{{end}}
{{define "user x"}}Пользователь {{.}}{{end}}
{{define "update"}}Обновить{{end}}
{{define "delete user"}}Удалить пользователя{{end}}
{{define "delete user tip"}}Удаляет пользователя из базы данных. Правки пользователя будут сохранены. Имя пользователя освободится для повторной регистрации.{{end}}

{{define "delete user?"}}Удалить пользователя {{.}}?{{end}}
{{define "delete user warning"}}Вы уверены, что хотите удалить этого пользователя из базы данных? Это действие нельзя отменить.{{end}}
`

func viewPanel(meta viewutil.Meta) {
	viewutil.ExecutePage(meta, panelChain, &viewutil.BaseData{})
}

type newUserData struct {
	*viewutil.BaseData
	Form util.FormData
	Groups []user.Group
}

func viewNewUser(meta viewutil.Meta, form util.FormData) {
	viewutil.ExecutePage(meta, newUserChain, newUserData{
		BaseData: &viewutil.BaseData{},
		Form:     form,
		Groups:   user.Groups(),
	})
}

type editDeleteUserData struct {
	*viewutil.BaseData
	Form util.FormData
	U    *user.User
	Groups []user.Group
}

func viewEditUser(meta viewutil.Meta, form util.FormData, u *user.User) {
	viewutil.ExecutePage(meta, editUserChain, editDeleteUserData{
		BaseData: &viewutil.BaseData{},
		Form:     form,
		U:        u,
		Groups:   user.Groups(),
	})
}

func viewDeleteUser(meta viewutil.Meta, form util.FormData, u *user.User) {
	viewutil.ExecutePage(meta, deleteUserChain, editDeleteUserData{
		BaseData: &viewutil.BaseData{},
		Form:     form,
		U:        u,
		Groups:   user.Groups(),
	})
}

// handlerAdmin provides the admin panel.
func handlerAdmin(w http.ResponseWriter, rq *http.Request) {
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.WriteHeader(http.StatusOK)
	viewPanel(viewutil.MetaFrom(w, rq))
}

// handlerAdminShutdown kills the wiki.
func handlerAdminShutdown(w http.ResponseWriter, rq *http.Request) {
	done := rq.Method == http.MethodPost
	_ = pageShutdown.RenderTo(viewutil.MetaFrom(w, rq), map[string]interface{}{
		"Done": done,
	})
	if done {
		slog.Info("An admin commanded the wiki to shutdown")
		process.Shutdown()
	}
}

// handlerAdminReindexHyphae reindexes all hyphae by checking the wiki storage directory anew.
func handlerAdminReindexHyphae(w http.ResponseWriter, rq *http.Request) {
	err := shroom.Reindex()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectTo := rq.Referer()
	if redirectTo == "" {
		redirectTo = cfg.Root + "admin"
	}
	http.Redirect(w, rq, redirectTo, http.StatusSeeOther)
}

// handlerAdminReindexUsers reinitialises the user system.
func handlerAdminReindexUsers(w http.ResponseWriter, rq *http.Request) {
	user.ReadUsersFromFilesystem()
	redirectTo := rq.Referer()
	if redirectTo == "" {
		redirectTo = cfg.Root + "users"
	}
	http.Redirect(w, rq, redirectTo, http.StatusSeeOther)
}

// handlerAdminUpdateHeaderLinks updates header links by reading the configured hypha, if there is any, or resorting to default values.
func handlerAdminUpdateHeaderLinks(w http.ResponseWriter, rq *http.Request) {
	slog.Info("Updating header links")
	shroom.SetHeaderLinks()
	redirectTo := rq.Referer()
	if redirectTo == "" {
		redirectTo = cfg.Root + "admin"
	}
	http.Redirect(w, rq, redirectTo, http.StatusSeeOther)
}

func handlerAdminUserEdit(w http.ResponseWriter, rq *http.Request) {
	vars := mux.Vars(rq)
	u := user.ByName(vars["username"])
	if u.IsEmpty() {
		util.HTTP404Page(w, "404 not found")
		return
	}
	f := util.FormDataFromRequest(rq, []string{"group"})

	if rq.Method == http.MethodPost {
		newGroup := f.Get("group")
		nu, err := u.WithGroupName(newGroup)
		if err != nil {
			f = f.WithError(err)
		} else if err = user.ReplaceUser(u, nu); err != nil {
			f = f.WithError(err)
		} else {
			http.Redirect(w, rq, cfg.Root + "users", http.StatusSeeOther)
			return
		}
	}

	f.Put("group", u.Group().Name())
	if f.HasError() {
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(".html"))
	viewEditUser(viewutil.MetaFrom(w, rq), f, u)
}

func handlerAdminUserChangePassword(w http.ResponseWriter, rq *http.Request) {
	vars := mux.Vars(rq)
	u := user.ByName(vars["username"])
	if u.IsEmpty() {
		util.HTTP404Page(w, "404 not found")
		return
	}

	f := util.FormDataFromRequest(rq, []string{"password", "password_confirm"})

	password := f.Get("password")
	passwordConfirm := f.Get("password_confirm")
	if password == passwordConfirm {
		nu, err := u.WithPassword(password)
		if err != nil {
			f = f.WithError(err)
		} else if err = user.ReplaceUser(u, nu); err != nil {
			f = f.WithError(err)
		} else {
			http.Redirect(w, rq, cfg.Root + "users", http.StatusSeeOther)
			return
		}
	} else {
		err := fmt.Errorf("passwords do not match")
		f = f.WithError(err)
	}

	if f.HasError() {
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(".html"))
	viewEditUser(viewutil.MetaFrom(w, rq), f, u)
}

func handlerAdminUserDelete(w http.ResponseWriter, rq *http.Request) {
	vars := mux.Vars(rq)
	u := user.ByName(vars["username"])
	if u.IsEmpty() {
		util.HTTP404Page(w, "404 page not found")
		return
	}

	f := util.NewFormData()

	if rq.Method == http.MethodPost {
		if err := user.DeleteUser(u.Name()); err != nil {
			slog.Info("Failed to delete user", "err", err)
			f = f.WithError(err)
		} else {
			http.Redirect(w, rq, cfg.Root + "users", http.StatusSeeOther)
			return
		}
	}

	if f.HasError() {
		w.WriteHeader(http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(".html"))
	viewDeleteUser(viewutil.MetaFrom(w, rq), f, u)
}

func handlerAdminUserNew(w http.ResponseWriter, rq *http.Request) {
	if rq.Method == http.MethodGet {
		w.Header().Set("Content-Type", mime.TypeByExtension(".html"))
		viewNewUser(viewutil.MetaFrom(w, rq), util.NewFormData())
	} else if rq.Method == http.MethodPost {
		// Create a user
		f := util.FormDataFromRequest(rq, []string{"name", "password", "group"})

		err := user.Register(
			f.Get("name"), f.Get("password"),
			f.Get("group"), "local", true,
		)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", mime.TypeByExtension(".html"))
			viewNewUser(viewutil.MetaFrom(w, rq), f.WithError(err))
		} else {
			http.Redirect(w, rq, cfg.Root + "users", http.StatusSeeOther)
		}
	}
}
