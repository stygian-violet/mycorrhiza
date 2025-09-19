package web

import (
	"html/template"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/hypview"
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/shroom"
	"github.com/bouncepaw/mycorrhiza/l18n"
	"github.com/bouncepaw/mycorrhiza/mycoopts"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"

	"git.sr.ht/~bouncepaw/mycomarkup/v5"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/mycocontext"
	"github.com/gorilla/mux"
)

func initMutators(r *mux.Router) {
	r.PathPrefix("/edit/").HandlerFunc(handlerEdit).Methods("GET")
	r.PathPrefix("/rename/").HandlerFunc(handlerRename).Methods("GET", "POST")
	r.PathPrefix("/delete/").HandlerFunc(handlerDelete).Methods("GET", "POST")
	r.PathPrefix("/revert/").HandlerFunc(handlerRevert).Methods("GET", "POST")
	r.PathPrefix("/remove-media/").HandlerFunc(handlerRemoveMedia).Methods("POST")
	r.PathPrefix("/upload-binary/").HandlerFunc(handlerUploadBinary).Methods("POST")
	r.PathPrefix("/upload-text/").HandlerFunc(handlerUploadText).Methods("POST")
}

/// TODO: this is no longer ridiculous, but is now ugly. Gotta make it at least bearable to look at :-/

func handlerRemoveMedia(w http.ResponseWriter, rq *http.Request) {
	util.PrepareRq(rq)
	var (
		h    = hyphae.ByName(util.HyphaNameFromRq(rq, "remove-media"))
		meta = viewutil.MetaFrom(w, rq)
	)
	if !meta.U.CanProceed("remove-media") {
		viewutil.HttpErr(meta, http.StatusForbidden, h.CanonicalName(), "Permission denied")
		return
	}
	switch h := h.(type) {
	case *hyphae.EmptyHypha, *hyphae.TextualHypha:
		viewutil.HttpErr(meta, http.StatusBadRequest, h.CanonicalName(), "No media to remove")
		return
	case *hyphae.MediaHypha:
		if err := shroom.RemoveMedia(meta.U, h); err != nil {
			slog.Error("Failed to remove media", "hypha", h, "err", err)
			viewutil.HttpErr(meta, http.StatusInternalServerError, h.CanonicalName(), err.Error())
			return
		}
	}
	http.Redirect(w, rq, cfg.Root+"hypha/"+h.CanonicalName(), http.StatusSeeOther)
}

func handlerDelete(w http.ResponseWriter, rq *http.Request) {
	util.PrepareRq(rq)
	var (
		h    = hyphae.ByName(util.HyphaNameFromRq(rq, "delete"))
		meta = viewutil.MetaFrom(w, rq)
	)

	if !meta.U.CanProceed("delete") {
		slog.Info("No permission to delete hypha",
			"user", meta.U, "hypha", h.CanonicalName())
		viewutil.HttpErr(meta, http.StatusForbidden, h.CanonicalName(), "Permission denied")
		return
	}

	switch h.(type) {
	case *hyphae.EmptyHypha:
		slog.Info("Trying to delete empty hyphae",
			"user", meta.U, "hypha", h.CanonicalName())
		// TODO: localize
		viewutil.HttpErr(meta, http.StatusBadRequest, h.CanonicalName(), "Cannot delete an empty hypha")
		return
	}

	if rq.Method == "GET" {
		_ = pageHyphaDelete.RenderTo(
			viewutil.MetaFrom(w, rq),
			map[string]any{
				"HyphaName": h.CanonicalName(),
			})
		return
	}

	recursive := rq.PostFormValue("recursive") == "true"
	if err := shroom.Delete(meta.U, h.(hyphae.ExistingHypha), recursive); err != nil {
		slog.Error("Failed to delete hypha", "hypha", h, "err", err)
		viewutil.HttpErr(meta, http.StatusInternalServerError, h.CanonicalName(), err.Error())
		return
	}
	http.Redirect(w, rq, cfg.Root+"hypha/"+h.CanonicalName(), http.StatusSeeOther)
}

func handlerRevert(w http.ResponseWriter, rq *http.Request) {
	util.PrepareRq(rq)

	shorterURL := strings.TrimPrefix(rq.URL.Path, cfg.Root+"revert/")
	revHash, slug, found := strings.Cut(shorterURL, "/")
	if !found || !util.IsRevHash(revHash) || len(slug) < 1 {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}

	var (
		hyphaName = util.CanonicalName(slug)
		h         = hyphae.ByName(hyphaName)
		meta      = viewutil.MetaFrom(w, rq)
	)

	if !meta.U.CanProceed("revert") {
		slog.Info(
			"No permission to revert hypha",
			"user", meta.U, "hyphaName", hyphaName,
		)
		viewutil.HttpErr(
			meta, http.StatusForbidden,
			hyphaName, "Permission denied",
		)
		return
	}

	if rq.Method == "GET" {
		_ = pageHyphaRevert.RenderTo(
			meta,
			map[string]any{
				"HyphaName": hyphaName,
				"RevHash": revHash,
			})
		return
	}

	h, err := shroom.Revert(meta.U, h, revHash)
	if err != nil {
		slog.Error("Failed to revert hypha", "err", err)
		viewutil.HttpErr(meta, http.StatusInternalServerError, h.CanonicalName(), err.Error())
		return
	}
	http.Redirect(w, rq, cfg.Root+"hypha/"+h.CanonicalName(), http.StatusSeeOther)
}

func handlerRename(w http.ResponseWriter, rq *http.Request) {
	util.PrepareRq(rq)
	var (
		lc   = l18n.FromRequest(rq)
		h    = hyphae.ByName(util.HyphaNameFromRq(rq, "rename"))
		meta = viewutil.MetaFrom(w, rq)
	)

	switch h.(type) {
	case *hyphae.EmptyHypha:
		slog.Info("Trying to rename empty hypha",
			"user", meta.U, "hypha", h.CanonicalName())
		viewutil.HttpErr(meta, http.StatusBadRequest, h.CanonicalName(), "Cannot rename an empty hypha") // TODO: localize
		return
	}

	if !meta.U.CanProceed("rename") {
		slog.Info("No permission to rename hypha",
			"user", meta.U, "hypha", h.CanonicalName())
		viewutil.HttpErr(meta, http.StatusForbidden, h.CanonicalName(), "Permission denied")
		return
	}

	var (
		oldHypha          = h.(hyphae.ExistingHypha)
		newName           = util.CanonicalName(rq.PostFormValue("new-name"))
		recursive         = rq.PostFormValue("recursive") == "true"
		leaveRedirections = rq.PostFormValue("redirection") == "true"
	)

	if rq.Method == "GET" {
		hypview.RenameHypha(meta, h.CanonicalName())
		return
	}

	if err := shroom.Rename(oldHypha, newName, recursive, leaveRedirections, meta.U); err != nil {
		slog.Error("Failed to rename hypha",
			"err", err, "user", meta.U, "hypha", oldHypha.CanonicalName())
		viewutil.HttpErr(meta, http.StatusForbidden, oldHypha.CanonicalName(), lc.Get(err.Error())) // TODO: localize
		return
	}
	http.Redirect(w, rq, cfg.Root+"hypha/"+newName, http.StatusSeeOther)
}

// handlerEdit shows the edit form. It doesn't edit anything actually.
func handlerEdit(w http.ResponseWriter, rq *http.Request) {
	util.PrepareRq(rq)

	var (
		lc   = l18n.FromRequest(rq)
		meta = viewutil.MetaFrom(w, rq)

		hyphaName = util.HyphaNameFromRq(rq, "edit")
		h         = hyphae.ByName(hyphaName)

		isNew   bool
		content string
		err     error
	)

	if !meta.U.CanProceed("upload-text") {
		viewutil.HttpErr(meta, http.StatusForbidden, hyphaName, "Permission denied")
		return
	}

	switch h.(type) {
	case *hyphae.EmptyHypha:
		isNew = true
	default:
		content, err = h.Text(history.FileReader())
		if err != nil {
			slog.Error("Failed to fetch Mycomarkup file", "err", err)
			viewutil.HttpErr(meta, http.StatusInternalServerError, hyphaName, lc.Get("ui.error_text_fetch"))
			return
		}
	}
	_ = pageHyphaEdit.RenderTo(
		viewutil.MetaFrom(w, rq),
		map[string]any{
			"HyphaName": hyphaName,
			"Content":   content,
			"IsNew":     isNew,
			"Message":   "",
			"Preview":   "",
		})
}

// handlerUploadText uploads a new text part for the hypha.
func handlerUploadText(w http.ResponseWriter, rq *http.Request) {
	util.PrepareRq(rq)

	var (
		meta      = viewutil.MetaFrom(w, rq)
		hyphaName = util.HyphaNameFromRq(rq, "upload-text")
		h         = hyphae.ByName(hyphaName)
		_, isNew  = h.(*hyphae.EmptyHypha)

		textData = rq.PostFormValue("text")
		action   = rq.PostFormValue("action")
		message  = rq.PostFormValue("message")
	)

	if !meta.U.CanProceed("upload-text") {
		viewutil.HttpErr(meta, http.StatusForbidden, hyphaName, "Permission denied")
		return
	}

	if action == "preview" {
		ctx, _ := mycocontext.ContextFromStringInput(textData, mycoopts.MarkupOptions(hyphaName))
		preview := template.HTML(mycomarkup.BlocksToHTML(ctx, mycomarkup.BlockTree(ctx)))

		_ = pageHyphaEdit.RenderTo(
			viewutil.MetaFrom(w, rq),
			map[string]any{
				"HyphaName": hyphaName,
				"Content":   textData,
				"IsNew":     isNew,
				"Message":   message,
				"Preview":   preview,
			})
		return
	}

	if err := shroom.UploadText(h, textData, message, meta.U); err != nil {
		viewutil.HttpErr(meta, http.StatusBadRequest, hyphaName, err.Error())
		return
	}
	http.Redirect(w, rq, cfg.Root+"hypha/"+hyphaName, http.StatusSeeOther)
}

// handlerUploadBinary uploads a new media for the hypha.
func handlerUploadBinary(w http.ResponseWriter, rq *http.Request) {
	util.PrepareRq(rq)

	hyphaName := util.HyphaNameFromRq(rq, "upload-binary")
	meta := viewutil.MetaFrom(w, rq)
	if !meta.U.CanProceed("upload-binary") {
		viewutil.HttpErr(meta, http.StatusForbidden, hyphaName, "Permission denied")
		return
	}

	file, header, err := rq.FormFile("binary")
	if err != nil {
		viewutil.HttpErr(meta, http.StatusBadRequest, hyphaName, err.Error())
		return
	}
	defer file.Close()

	h := hyphae.ByName(hyphaName)
	mime := header.Header.Get("Content-Type")

	if err := shroom.UploadBinary(h, header.Filename, mime, file, meta.U); err != nil {
		viewutil.HttpErr(meta, http.StatusInternalServerError, hyphaName, err.Error())
		return
	}
	http.Redirect(w, rq, cfg.Root+"hypha/"+hyphaName, http.StatusSeeOther)
}
