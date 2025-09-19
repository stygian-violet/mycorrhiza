// Package misc provides miscellaneous informative views.
package misc

import (
	"io"
	"log/slog"
	"math/rand"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/search"
	"github.com/bouncepaw/mycorrhiza/internal/shroom"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/l18n"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/static"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

func InitAssetHandlers(rtr *mux.Router) {
	rtr.HandleFunc("/static/style.css", handlerStyle)
	rtr.HandleFunc("/robots.txt", handlerRobotsTxt)
	rtr.PathPrefix("/static/").
		Handler(http.StripPrefix(cfg.Root + "static/", http.FileServer(http.FS(static.FS))))
	rtr.HandleFunc("/favicon.ico", func(w http.ResponseWriter, rq *http.Request) {
		http.Redirect(w, rq, cfg.Root + "static/favicon.ico", http.StatusSeeOther)
	})
}

func InitHandlers(rtr *mux.Router) {
	rtr.HandleFunc("/list", handlerList)
	rtr.HandleFunc("/reindex", handlerReindex)
	rtr.HandleFunc("/update-header-links", handlerUpdateHeaderLinks)
	rtr.HandleFunc("/random", handlerRandom)
	rtr.HandleFunc("/about", handlerAbout)
	rtr.HandleFunc("/title-search/", handlerTitleSearch)
	if cfg.FullTextSearchPage {
		rtr.HandleFunc("/text-search/", handlerTextSearch)
	}
	initViews()
}

// handlerList shows a list of all hyphae in the wiki in random order.
func handlerList(w http.ResponseWriter, rq *http.Request) {
	// TODO: make this more effective, there are too many loops and vars
	var (
		sortedHypha = hyphae.PathographicSort(hyphae.YieldExistingHyphaNames())
		entries     []listDatum
	)
	for hyphaName := range sortedHypha {
		switch h := hyphae.ByName(hyphaName).(type) {
		case *hyphae.TextualHypha:
			entries = append(entries, listDatum{hyphaName, ""})
		case *hyphae.MediaHypha:
			entries = append(entries, listDatum{hyphaName, filepath.Ext(h.MediaFilePath())[1:]})
		}
	}
	viewList(viewutil.MetaFrom(w, rq), entries)
}

// handlerReindex reindexes all hyphae by checking the wiki storage directory anew.
func handlerReindex(w http.ResponseWriter, rq *http.Request) {
	if ok := user.CanProceed(rq, "reindex"); !ok {
		var lc = l18n.FromRequest(rq)
		viewutil.HttpErr(viewutil.MetaFrom(w, rq), http.StatusForbidden, cfg.HomeHypha, lc.Get("ui.reindex_no_rights"))
		slog.Info("No rights to reindex")
		return
	}
	shroom.Reindex()
	http.Redirect(w, rq, cfg.Root, http.StatusSeeOther)
}

// handlerUpdateHeaderLinks updates header links by reading the configured hypha, if there is any, or resorting to default values.
func handlerUpdateHeaderLinks(w http.ResponseWriter, rq *http.Request) {
	if ok := user.CanProceed(rq, "update-header-links"); !ok {
		var lc = l18n.FromRequest(rq)
		viewutil.HttpErr(viewutil.MetaFrom(w, rq), http.StatusForbidden, cfg.HomeHypha, lc.Get("ui.header_no_rights"))
		slog.Info("No rights to update header links")
		return
	}
	slog.Info("Updated header links")
	shroom.SetHeaderLinks()
	http.Redirect(w, rq, cfg.Root, http.StatusSeeOther)
}

// handlerRandom redirects to a random hypha.
func handlerRandom(w http.ResponseWriter, rq *http.Request) {
	var (
		randomHyphaName string
		amountOfHyphae  = hyphae.Count()
	)
	if amountOfHyphae == 0 {
		var lc = l18n.FromRequest(rq)
		viewutil.HttpErr(viewutil.MetaFrom(w, rq), http.StatusNotFound, cfg.HomeHypha, lc.Get("ui.random_no_hyphae_tip"))
		return
	}
	i := rand.Intn(amountOfHyphae)
	for h := range hyphae.YieldExistingHyphae() {
		if i == 0 {
			randomHyphaName = h.CanonicalName()
			break
		}
		i--
	}
	http.Redirect(w, rq, cfg.Root+"hypha/"+randomHyphaName, http.StatusSeeOther)
}

// handlerAbout shows a summary of wiki's software.
func handlerAbout(w http.ResponseWriter, rq *http.Request) {
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.WriteHeader(http.StatusOK)
	var (
		lc    = l18n.FromRequest(rq)
		title = lc.Get("ui.about_title", &l18n.Replacements{"name": cfg.WikiName})
	)
	_, err := io.WriteString(w, viewutil.Base(
		viewutil.MetaFrom(w, rq),
		title,
		AboutHTML(lc),
		map[string]string{},
	))
	if err != nil {
		slog.Error("Failed to write About template", "err", err)
	}
}

var stylesheets = []string{"default.css", "custom.css"}

func handlerStyle(w http.ResponseWriter, rq *http.Request) {
	w.Header().Set("Content-Type", mime.TypeByExtension(".css"))
	for _, name := range stylesheets {
		file, err := static.FS.Open(name)
		if err != nil {
			continue
		}
		_, err = io.Copy(w, file)
		if err != nil {
			slog.Error("Failed to write stylesheet; proceeding anyway", "err", err)
		}
		_ = file.Close()
	}
}

func handlerRobotsTxt(w http.ResponseWriter, rq *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	file, err := static.FS.Open("robots.txt")
	if err != nil {
		return
	}
	_, err = io.Copy(w, file)
	if err != nil {
		slog.Error("Failed to write robots.txt; proceeding anyway", "err", err)
	}
	_ = file.Close()
}

func handlerTitleSearch(w http.ResponseWriter, rq *http.Request) {
	_ = rq.ParseForm()
	var (
		meta        = viewutil.MetaFrom(w, rq)
		query       = normalizeQuery(rq.FormValue("q"))
		hyphaName   = util.CanonicalName(query)
		_, nameFree = hyphae.AreFreeNames(hyphaName)
		results     []string
		textResults *search.SearchResults = nil
	)
	if query != "" {
		for hyphaName := range shroom.YieldHyphaNamesContainingString(query) {
			results = append(results, hyphaName)
		}
		if (cfg.FullTextSearch != cfg.FullTextDisabled &&
			cfg.FullTextLowerLimit != 0 &&
			meta.U.CanProceed("text-search")) {
			textResults, _ = fullTextSearch(query, cfg.FullTextLowerLimit)
		}
	}
	w.WriteHeader(http.StatusOK)
	viewTitleSearch(meta, query, hyphaName, !nameFree, results, textResults)
}

func handlerTextSearch(w http.ResponseWriter, rq *http.Request) {
	_ = rq.ParseForm()
	meta := viewutil.MetaFrom(w, rq)
	if !meta.U.CanProceed("text-search") {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, "403 Forbidden")
		return
	}
	if cfg.FullTextSearch == cfg.FullTextDisabled || cfg.FullTextUpperLimit == 0 {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "404 Not found")
		return
	}
	var (
		query = normalizeQuery(rq.FormValue("q"))
		results *search.SearchResults = nil
		err error = nil
	)
	if query != "" {
		results, err = fullTextSearch(query, cfg.FullTextUpperLimit)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	viewTextSearch(meta, query, results)
}
