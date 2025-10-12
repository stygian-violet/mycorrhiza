// Package web contains web handlers and initialization stuff.
package web

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/bouncepaw/mycorrhiza/help"
	"github.com/bouncepaw/mycorrhiza/history/histweb"
	"github.com/bouncepaw/mycorrhiza/hypview"
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/interwiki"
	"github.com/bouncepaw/mycorrhiza/l18n"
	"github.com/bouncepaw/mycorrhiza/misc"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"

	"github.com/gorilla/mux"
)

// Handler initializes and returns the HTTP router based on the configuration.
func Handler() *mux.Router {
	router := mux.NewRouter()
	ret := router
	if cfg.Root != "/" {
		router = router.PathPrefix(strings.TrimSuffix(cfg.Root, "/")).Subrouter()
	}

	router.Use(baseMiddleware)
	router.StrictSlash(true)

	// Public routes. They're always accessible regardless of the user status.
	misc.InitAssetHandlers(router)

	r := router.PathPrefix("").Subrouter()
	r.Use(authMiddleware)
	// Auth
	// The check below saves a lot of extra checks and lines of codes in other places in this file.
	if cfg.UseAuth {
		if cfg.AllowRegistration {
			r.HandleFunc("/register", handlerRegister).Methods(http.MethodPost, http.MethodGet)
		}
		if cfg.TelegramEnabled {
			r.HandleFunc("/telegram-login", handlerTelegramLogin).Methods(http.MethodPost, http.MethodGet)
		}
		r.HandleFunc("/login", handlerLogin).Methods(http.MethodPost, http.MethodGet)
		r.HandleFunc("/logout", handlerLogout).Methods(http.MethodPost)
	}

	// Wiki routes. They may be locked or restricted.
	r = router.PathPrefix("").Subrouter()
	r.Use(wikiMiddleware)

	initReaders(r)
	initMutators(r)
	help.InitHandlers(r)
	misc.InitHandlers(r)
	hypview.Init()
	histweb.InitHandlers(r)
	interwiki.InitHandlers(r)

	r.PathPrefix("/add-to-category").HandlerFunc(handlerAddToCategory).Methods("POST")
	r.PathPrefix("/remove-from-category").HandlerFunc(handlerRemoveFromCategory).Methods("POST")
	r.PathPrefix("/category/").HandlerFunc(handlerCategory).Methods("GET")
	r.PathPrefix("/edit-category/").HandlerFunc(handlerEditCategory).Methods("GET")
	r.PathPrefix("/category").HandlerFunc(handlerListCategory).Methods("GET")

	// Admin routes
	if cfg.UseAuth {
		r.HandleFunc("/users", handlerUserList).Methods(http.MethodGet)

		adminRouter := r.PathPrefix("/admin").Subrouter()

		adminRouter.HandleFunc("/shutdown", handlerAdminShutdown).Methods(http.MethodPost)
		adminRouter.HandleFunc("/reindex-users", handlerAdminReindexUsers).Methods(http.MethodPost)
		adminRouter.HandleFunc("/reindex-hyphae", handlerAdminReindexHyphae).Methods(http.MethodPost)
		adminRouter.HandleFunc("/update-header-links", handlerAdminUpdateHeaderLinks).Methods(http.MethodPost)

		adminRouter.HandleFunc("/new-user", handlerAdminUserNew).Methods(http.MethodGet, http.MethodPost)
		adminRouter.HandleFunc("/users/{username}/edit", handlerAdminUserEdit).Methods(http.MethodGet, http.MethodPost)
		adminRouter.HandleFunc("/users/{username}/change-password", handlerAdminUserChangePassword).Methods(http.MethodPost)
		adminRouter.HandleFunc("/users/{username}/delete", handlerAdminUserDelete).Methods(http.MethodGet, http.MethodPost)

		adminRouter.HandleFunc("/", handlerAdmin).Methods("GET")

		settingsRouter := r.PathPrefix("/settings").Subrouter()
		// TODO: check if necessary?
		settingsRouter.HandleFunc("/change-password", handlerUserChangePassword).Methods(http.MethodGet, http.MethodPost)
	}

	// Index page
	r.HandleFunc("/", func(w http.ResponseWriter, rq *http.Request) {
		addr, err := url.Parse(cfg.Root + "hypha/" + cfg.HomeHypha)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rq.URL = addr
		handlerHypha(w, rq)
	})

	initPages()

	return ret
}

// Auth

// handlerRegister displays the register form (GET) or registers the user (POST).
func handlerRegister(w http.ResponseWriter, rq *http.Request) {
	registerAnonOnLocked := cfg.Locked && cfg.RegistrationGroup == "anon"
	if rq.Method == http.MethodGet {
		_ = pageAuthRegister.RenderTo(viewutil.MetaFrom(w, rq), map[string]any{
			"RawQuery":             rq.URL.RawQuery,
			"RegisterAnonOnLocked": registerAnonOnLocked,
		})
		return
	}
	var (
		username = rq.PostFormValue("username")
		password = rq.PostFormValue("password")
		err      = user.Register(username, password, cfg.RegistrationGroup, "local", false)
	)
	if err != nil {
		slog.Info("Failed to register", "username", username, "err", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		_ = pageAuthRegister.RenderTo(viewutil.MetaFrom(w, rq), map[string]any{
			"RawQuery":             rq.URL.RawQuery,
			"Err":                  err,
			"Username":             username,
			"Password":             password,
			"RegisterAnonOnLocked": registerAnonOnLocked,
		})
		return
	}
	slog.Info("Registered user", "username", username)
	err = user.LoginDataHTTP(w, username, password)
	if err != nil {
		meta := viewutil.MetaFrom(w, rq)
		_ = pageAuthLogin.RenderTo(meta, map[string]any{
			"AllowRegistration": true,
			"Locked":            meta.U.ShowLock(),
			"Err":               err,
			"Username":          username,
		})
		return
	}
	http.Redirect(w, rq, cfg.Root+rq.URL.RawQuery, http.StatusSeeOther)
}

// handlerLogout shows the logout form (GET) or logs the user out (POST).
func handlerLogout(w http.ResponseWriter, rq *http.Request) {
	user.LogoutFromRequest(w, rq)
	http.Redirect(w, rq, cfg.Root, http.StatusSeeOther)
}

// handlerLogin shows the login form (GET) or logs the user in (POST).
func handlerLogin(w http.ResponseWriter, rq *http.Request) {
	meta := viewutil.MetaFrom(w, rq)
	locked := meta.U.ShowLock()

	if rq.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		_ = pageAuthLogin.RenderTo(meta, map[string]any{
			"AllowRegistration": cfg.AllowRegistration,
			"Locked":            locked,
			"WikiName":          cfg.WikiName,
		})
		return
	}

	var (
		username = util.CanonicalName(rq.PostFormValue("username"))
		password = rq.PostFormValue("password")
		err      = user.LoginDataHTTP(w, username, password)
	)
	if err != nil {
		_ = pageAuthLogin.RenderTo(meta, map[string]any{
			"AllowRegistration": cfg.AllowRegistration,
			"ErrLogin":    errors.Is(err, user.ErrLogin),
			"ErrTelegram": false, // TODO: ?
			"Err":         err.Error(),
			"Locked":      locked,
			"WikiName":    cfg.WikiName,
			"Username":    username,
		})
		slog.Info("Failed to log in", "username", username, "err", err.Error())
		return
	}
	http.Redirect(w, rq, cfg.Root, http.StatusSeeOther)
	slog.Info("Logged in", "username", username)
}

func handlerTelegramLogin(w http.ResponseWriter, rq *http.Request) {
	// Note there is no lock here.
	lc := l18n.FromRequest(rq)
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	_ = rq.ParseForm()
	var (
		values     = rq.URL.Query()
		username   = strings.ToLower(values.Get("username"))
		seemsValid = user.TelegramAuthParamsAreValid(values)
		err        = user.Register(
			username,
			"", // Password matters not
			cfg.RegistrationGroup,
			"telegram",
			false,
		)
	)
	// If registering a user via Telegram failed, because a Telegram user with this name
	// has already registered, then everything is actually ok!
	if user.ByName(username).Source() == user.UserSourceTelegram {
		err = nil
	}

	if !seemsValid {
		err = errors.New("Wrong parameters")
	}

	if err != nil {
		slog.Info("Failed to register", "username", username, "err", err.Error(), "method", "telegram")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(
			w,
			viewutil.Base(
				viewutil.MetaFrom(w, rq),
				lc.Get("ui.error"),
				fmt.Sprintf(
					`<main class="main-width"><p>%s</p><p>%s</p><p><a href="%slogin">%s<a></p></main>`,
					lc.Get("auth.error_telegram"),
					err.Error(),
					cfg.Root,
					lc.Get("auth.go_login"),
				),
				map[string]string{},
			),
		)
		return
	}

	errmsg := user.LoginDataHTTP(w, username, "")
	if errmsg != nil {
		slog.Error("Failed to login using Telegram", "err", err, "username", username)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(
			w,
			viewutil.Base(
				viewutil.MetaFrom(w, rq),
				"Error",
				fmt.Sprintf(
					`<main class="main-width"><p>%s</p><p>%s</p><p><a href="%slogin">%s<a></p></main>`,
					lc.Get("auth.error_telegram"),
					err.Error(),
					cfg.Root,
					lc.Get("auth.go_login"),
				),
				map[string]string{},
			),
		)
		return
	}
	http.Redirect(w, rq, cfg.Root, http.StatusSeeOther)
	slog.Info("Logged in", "username", username, "method", "telegram")
}
