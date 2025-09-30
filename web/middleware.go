package web

import (
	// "log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/user"
)

func maxBodySize(rq *http.Request) int64 {
	path := rq.URL.Path
	var fileSize int64
	switch {
	case strings.HasPrefix(path, cfg.Root + "upload-text/"):
		fileSize = cfg.MaxTextSize
	case strings.HasPrefix(path, cfg.Root + "upload-binary/"):
		fileSize = cfg.MaxMediaSize
	default:
		return cfg.MaxFormSize
	}
	if fileSize == 0 {
		return 0
	}
	return cfg.MaxFormSize + fileSize
}

func parseForm(rq *http.Request) error {
	switch {
	case rq.Method != http.MethodPost:
		return nil
	case rq.Header.Get("Content-type") == "application/x-www-form-urlencoded":
		return rq.ParseForm()
	default:
		return rq.ParseMultipartForm(1 << 10)
	}
}

func baseMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		// slog.Info("baseMiddleware", "path", rq.URL.Path, "method", rq.Method)
		p := path.Clean(rq.URL.Path)
		if p == "/" || p == "." {
			p = ""
		}
		rq.URL.Path = p
		w.Header().Add("Content-Security-Policy", cfg.CSP)
		w.Header().Add("Referrer-Policy", cfg.Referrer)
		next.ServeHTTP(w, rq)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		// slog.Info("authMiddleware", "path", rq.URL.Path, "method", rq.Method)
		rq.Body = http.MaxBytesReader(w, rq.Body, cfg.MaxFormSize)
		if err := parseForm(rq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, rq)
		if rq.MultipartForm != nil {
			rq.MultipartForm.RemoveAll()
		}
	})
}

func wikiMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, rq *http.Request) {
		user := user.FromRequest(rq)
		// slog.Info("wikiMiddleware", "path", rq.URL.Path, "method", rq.Method, "user", user)
		if user.ShowLock() {
			http.Redirect(w, rq, cfg.Root + "lock", http.StatusSeeOther)
			return
		}
		route := rq.URL.Path
		if strings.HasPrefix(route, cfg.Root) {
			route = rq.URL.Path[len(cfg.Root):]
		} else if strings.HasPrefix(cfg.Root, route) {
			route = ""
		} else {
			http.Error(w, "400 Bad Request", http.StatusBadRequest)
			return
		}
		if route == "" {
			route = "hypha/" + cfg.HomeHypha
		}
		if !user.CanProceed(route) {
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}
		maxSize := maxBodySize(rq)
		if maxSize > 0 {
			rq.Body = http.MaxBytesReader(w, rq.Body, maxSize)
		}
		if err := parseForm(rq); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, rq)
		if rq.MultipartForm != nil {
			rq.MultipartForm.RemoveAll()
		}
	})
}
