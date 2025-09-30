package web

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/categories"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

func handlerEditCategory(w http.ResponseWriter, rq *http.Request) {
	meta := viewutil.MetaFrom(w, rq)
	catName := util.CanonicalName(strings.TrimPrefix(strings.TrimPrefix(rq.URL.Path, cfg.Root + "edit-category"), "/"))
	if catName == "" {
		viewutil.HandlerNotFound(w, rq)
		return
	}

	slog.Info("Editing category", "name", catName)
	_ = pageCatEdit.RenderTo(meta, map[string]any{
		"Addr":                    cfg.Root + "edit-category/" + catName,
		"CatName":                 catName,
		"Hyphae":                  categories.HyphaeInCategory(catName),
		"GivenPermissionToModify": meta.U.CanProceed("add-to-category"),
	})
}

func handlerListCategory(w http.ResponseWriter, rq *http.Request) {
	slog.Info("Viewing list of categories")
	cats := categories.ListOfCategories()
	sort.Strings(cats)

	_ = pageCatList.RenderTo(viewutil.MetaFrom(w, rq), map[string]any{
		"Addr":       cfg.Root + "category",
		"Categories": cats,
	})
}

func handlerCategory(w http.ResponseWriter, rq *http.Request) {
	catName := util.CanonicalName(strings.TrimPrefix(strings.TrimPrefix(rq.URL.Path, cfg.Root + "category"), "/"))
	if catName == "" {
		handlerListCategory(w, rq)
		return
	}

	meta := viewutil.MetaFrom(w, rq)
	slog.Info("Viewing category", "name", catName)
	_ = pageCatPage.RenderTo(meta, map[string]any{
		"Addr":                    cfg.Root + "category/" + catName,
		"CatName":                 catName,
		"Hyphae":                  categories.HyphaeInCategory(catName),
		"GivenPermissionToModify": meta.U.CanProceed("add-to-category"),
	})
}

// A request for removal of hyphae can either remove one hypha (used in the card on /hypha) or many hyphae (used in /edit-category). Both approaches are handled by /remove-from-category. This function finds all passed hyphae.
//
// There is one hypha from the hypha field. Then there are n hyphae in fields prefixed by _. It seems like I have to do it myself. Compare with PHP which handles it for you. I hope I am doing this wrong.
func hyphaeFromRequest(rq *http.Request) (canonicalNames []string) {
	if err := rq.ParseForm(); err != nil {
		slog.Info("Failed to parse form", "err", err)
	}
	if hyphaName := util.CanonicalName(rq.PostFormValue("hypha")); hyphaName != "" {
		canonicalNames = append(canonicalNames, hyphaName)
	}
	// According to https://developer.mozilla.org/en-US/docs/Web/HTML/Element/input/checkbox,
	//
	// > If a checkbox is unchecked when its form is submitted, there is no value submitted to the server to represent its unchecked state
	//
	// It means, that if there is a value, it is checked. And we can ignore values altogether.
	for key, _ := range rq.PostForm {
		// No prefix or "_":
		if !strings.HasPrefix(key, "_") || len(key) == 1 {
			continue
		}
		canonicalNames = append(canonicalNames, util.CanonicalName(key[1:]))
	}
	return
}

func handlerRemoveFromCategory(w http.ResponseWriter, rq *http.Request) {
	var (
		u          = user.FromRequest(rq)
		hyphaNames = hyphaeFromRequest(rq)
		catName    = util.CanonicalName(rq.PostFormValue("cat"))
		redirectTo = rq.PostFormValue("redirect-to")
	)
	if len(hyphaNames) == 0 || catName == "" {
		slog.Info("No data for removal of hyphae from category passed",
			"user", u, "catName", catName)
		http.Redirect(w, rq, redirectTo, http.StatusSeeOther)
		return
	}
	categories.RemoveHyphaeFromCategory(catName, hyphaNames...)
	slog.Info("Remove hyphae from category",
		"user", u, "catName", catName, "hyphaNames", hyphaNames)
	http.Redirect(w, rq, redirectTo, http.StatusSeeOther)
}

func handlerAddToCategory(w http.ResponseWriter, rq *http.Request) {
	var (
		hyphaName  = util.CanonicalName(rq.PostFormValue("hypha"))
		catName    = util.CanonicalName(rq.PostFormValue("cat"))
		redirectTo = rq.PostFormValue("redirect-to")
		u          = user.FromRequest(rq)
	)
	if hyphaName == "" || catName == "" {
		http.Redirect(w, rq, redirectTo, http.StatusSeeOther)
		return
	}
	slog.Info("Add hypha to category",
		"user", u, "catName", catName, "hyphaName", hyphaName)
	categories.AddHyphaeToCategory(catName, hyphaName)
	http.Redirect(w, rq, redirectTo, http.StatusSeeOther)
}
