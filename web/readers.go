package web

import (
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/hypview"
	"github.com/bouncepaw/mycorrhiza/internal/categories"
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/mimetype"
	"github.com/bouncepaw/mycorrhiza/internal/tree"
	"github.com/bouncepaw/mycorrhiza/l18n"
	"github.com/bouncepaw/mycorrhiza/mycoopts"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"

	"git.sr.ht/~bouncepaw/mycomarkup/v5"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/mycocontext"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/tools"
	"github.com/gorilla/mux"
)

func initReaders(r *mux.Router) {
	r.PathPrefix("/hypha/").HandlerFunc(handlerHypha).Methods("GET")
	r.PathPrefix("/text/").HandlerFunc(handlerText).Methods("GET")
	r.PathPrefix("/binary/").HandlerFunc(handlerBinary).Methods("GET")
	r.PathPrefix("/rev/").HandlerFunc(handlerRevision).Methods("GET")
	r.PathPrefix("/rev-text/").HandlerFunc(handlerRevisionText).Methods("GET")
	r.PathPrefix("/rev-binary/").HandlerFunc(handlerRevisionBinary).Methods("GET")
	r.PathPrefix("/media/").HandlerFunc(handlerMedia).Methods("GET")
	r.Path("/today").HandlerFunc(handlerToday).Methods("GET")
	r.Path("/edit-today").HandlerFunc(handlerEditToday).Methods("GET")

	// Backlinks
	r.PathPrefix("/backlinks/").HandlerFunc(handlerBacklinks).Methods("GET")
	r.PathPrefix("/orphans").HandlerFunc(handlerOrphans).Methods("GET")
	r.PathPrefix("/subhyphae/").HandlerFunc(handlerSubhyphae).Methods("GET")
}

func handlerEditToday(w http.ResponseWriter, rq *http.Request) {
	today := time.Now().Format(time.DateOnly)
	http.Redirect(w, rq, cfg.Root+"edit/"+today, http.StatusSeeOther)
}

func handlerToday(w http.ResponseWriter, rq *http.Request) {
	today := time.Now().Format(time.DateOnly)
	http.Redirect(w, rq, cfg.Root+"hypha/"+today, http.StatusSeeOther)
}

func handlerMedia(w http.ResponseWriter, rq *http.Request) {
	var (
		hyphaName = util.HyphaNameFromRq(rq, "media")
		h         = hyphae.ByName(hyphaName)
		meta      = viewutil.MetaFrom(w, rq)
		isMedia   = false

		upload    = path.Join("upload-binary", h.CanonicalName())
		remove    = path.Join("remove-media", h.CanonicalName())

		mime     string
		fileSize int64
		fileName string
	)
	switch h := h.(type) {
	case *hyphae.MediaHypha:
		isMedia = true
		mime = mimetype.FromExtension(path.Ext(h.MediaFilePath()))

		fileinfo, err := os.Stat(h.MediaFilePath())
		if err != nil {
			slog.Error("failed to stat media file", "err", err)
			// no return
		}

		fileSize = fileinfo.Size()
		fileName = path.Base(h.MediaFilePath())
	}
	_ = pageMedia.RenderTo(meta, map[string]any{
		"CanUpload":    meta.U.CanProceed(upload),
		"CanRemove":    isMedia && meta.U.CanProceed(remove),
		"HyphaName":    h.CanonicalName(),
		"IsMediaHypha": isMedia,
		"MimeType":     mime,
		"FileSize":     fileSize,
		"FileName":     fileName,
	})
}

// handlerRevisionText sends Mycomarkup text of the hypha at the given revision. See also: handlerRevision, handlerText.
//
// /rev-text/<revHash>/<hyphaName>
func handlerRevisionText(w http.ResponseWriter, rq *http.Request) {
	shorterURL := strings.TrimPrefix(rq.URL.Path, cfg.Root+"rev-text/")
	revHash, slug, found := strings.Cut(shorterURL, "/")
	if !found || !util.IsRevHash(revHash) || len(slug) < 1 {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}
	var (
		hyphaName = util.CanonicalName(slug)
		h         = hyphae.ByName(hyphaName)
	)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	mycoFilePath := h.TextFilePath()
	var textContents, err = history.FileAtRevision(mycoFilePath, revHash)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		slog.Error("Failed to serve text part",
			"err", err, "hyphaName", hyphaName, "revHash", revHash)
		_, _ = io.WriteString(w, "Error: "+err.Error())
		return
	}

	slog.Info("Serving text part",
		"hyphaName", hyphaName, "revHash", revHash, "mycoFilePath", mycoFilePath)
	w.WriteHeader(http.StatusOK)
	w.Write(textContents)
}

// handlerRevisionBinary sends hypha media at the given revision. See also: handlerRevision, handlerBinary.
//
// /rev-binary/<revHash>/<hyphaName>
func handlerRevisionBinary(w http.ResponseWriter, rq *http.Request) {
	shorterURL := strings.TrimPrefix(rq.URL.Path, cfg.Root + "rev-binary/")
	revHash, slug, found := strings.Cut(shorterURL, "/")
	if !found || !util.IsRevHash(revHash) || len(slug) < 1 {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}
	hyphaName := util.CanonicalName(slug)

	path, size, err := history.MediaAtRevision(hyphaName, revHash)
	switch {
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error(
			"Failed to find media file",
			"err", err, "hyphaName", hyphaName, "revHash", revHash,
		)
		_, _ = io.WriteString(w, "Error: " + err.Error())
		return
	case path == "":
		http.Error(w, "404 not found", http.StatusNotFound)
		return
	}

	slog.Info("Serving media file", "path", path)
	w.Header().Set("Content-Type", mimetype.FromExtension(filepath.Ext(path)))
	w.Header().Set("Content-Length", strconv.FormatUint(size, 10))
	filename := filepath.Base(path)
	filename = fmt.Sprintf(
		`attachment; filename="%s"; filename*=UTF-8''%s`,
		filename,
		url.QueryEscape(filename),
	)
	w.Header().Set("Content-Disposition", filename)

	cmd, contents, err := history.OpenFileAtRevision(path, revHash)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		slog.Error(
			"Failed to open media file",
			"err", err, "hyphaName", hyphaName,
			"revHash", revHash, "path", path,
		)
		_, _ = io.WriteString(w, "Error: " + err.Error())
		return
	}
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, contents)
	contents.Close()
	if err != nil {
		slog.Error(
			"Failed to serve media file",
			"err", err, "hyphaName", hyphaName,
			"revHash", revHash, "path", path,
		)
	}
	err = cmd.Wait()
	if err != nil {
		slog.Error(
			"Failed to serve media file",
			"err", err, "hyphaName", hyphaName,
			"revHash", revHash, "path", path,
		)
	}
}

// handlerRevision displays a specific revision of the hypha
func handlerRevision(w http.ResponseWriter, rq *http.Request) {
	lc := l18n.FromRequest(rq)
	shorterURL := strings.TrimPrefix(rq.URL.Path, cfg.Root+"rev/")
	revHash, slug, found := strings.Cut(shorterURL, "/")
	if !found || !util.IsRevHash(revHash) || len(slug) < 1 {
		http.Error(w, "400 bad request", http.StatusBadRequest)
		return
	}
	var (
		hyphaName     = util.CanonicalName(slug)
		h             = hyphae.ByName(hyphaName)
		contents      = template.HTML(fmt.Sprintf(`<p>%s</p>`, lc.Get("ui.revision_no_text")))
		textContents  []byte
		err           error
		mycoFilePath  = h.TextFilePath()
		mediaFilePath string
	)

	mediaFilePath, _, err = history.MediaAtRevision(hyphaName, revHash)
	if err != nil {
		slog.Error(
			"Failed to find media file",
			"err", err, "hyphaName", hyphaName, "revHash", revHash,
		)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "Error: " + err.Error())
		return
	}

	textContents, err = history.FileAtRevision(mycoFilePath, revHash)
	if err == nil {
		ctx, _ := mycocontext.ContextFromBytes(
			textContents,
			mycoopts.MarkupOptions(hyphaName),
		)
		contents = template.HTML(mycomarkup.BlocksToHTML(ctx, mycomarkup.BlockTree(ctx)))
	}
	if mediaFilePath != "" {
		mediaFileUrl := fmt.Sprintf(
			"%srev-binary/%s/%s",
			cfg.Root, revHash, hyphaName,
		)
		contents = template.HTML(
			mycoopts.MediaFile(mediaFilePath, mediaFileUrl, lc),
		) + contents
	}

	meta := viewutil.MetaFrom(w, rq)
	_ = pageRevision.RenderTo(meta, map[string]any{
		"ViewScripts": cfg.ViewScripts,
		"Contents":    contents,
		"RevHash":     revHash,
		"NaviTitle":   hypview.NaviTitle(meta, h.CanonicalName()),
		"HyphaName":   h.CanonicalName(),
		"CanRevert":   meta.U.CanProceed(path.Join("revert", h.CanonicalName())),
	})
}

// handlerText serves raw source text of the hypha.
func handlerText(w http.ResponseWriter, rq *http.Request) {
	hyphaName := util.HyphaNameFromRq(rq, "text")
	if h, ok := hyphae.ByName(hyphaName).(hyphae.ExistingHypha); ok {
		slog.Info("Serving text part", "path", h.TextFilePath())
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		history.FileReader().ServeFile(w, rq, h.TextFilePath())
	}
}

// handlerBinary serves attachment of the hypha.
func handlerBinary(w http.ResponseWriter, rq *http.Request) {
	hyphaName := util.HyphaNameFromRq(rq, "binary")
	switch h := hyphae.ByName(hyphaName).(type) {
	case *hyphae.EmptyHypha, *hyphae.TextualHypha:
		w.WriteHeader(http.StatusNotFound)
		slog.Info("Textual hypha has no media file; cannot serve it",
			"hyphaName", h.CanonicalName())
	case *hyphae.MediaHypha:
		path := h.MediaFilePath()
		filename := filepath.Base(path)
		slog.Info("Serving media file", "path", path)
		w.Header().Set("Content-Type", mimetype.FromExtension(filepath.Ext(path)))
		filename = fmt.Sprintf(
			`attachment; filename="%s"; filename*=UTF-8''%s`,
			filename,
			url.QueryEscape(filename),
		)
		w.Header().Set("Content-Disposition", filename)
		history.FileReader().ServeFile(w, rq, path)
	}
}

// handlerHypha is the main hypha action that displays the hypha and the binary upload form along with some navigation.
func handlerHypha(w http.ResponseWriter, rq *http.Request) {
	meta := viewutil.MetaFrom(w, rq)
	username := meta.U.Name()
	var (
		hyphaName     = util.HyphaNameFromRq(rq, "hypha")
		h             = hyphae.ByName(hyphaName)
		contents      template.HTML
		openGraph     template.HTML
		lc            = l18n.FromRequest(rq)
		cats          = categories.CategoriesWithHypha(h.CanonicalName())
		category_list = ":" + strings.Join(cats, ":") + ":"
		isMyProfile   = cfg.UseAuth && !meta.U.IsEmpty() && util.IsProfileName(h.CanonicalName()) && username == strings.TrimPrefix(h.CanonicalName(), cfg.UserHypha+"/")

		subhyphae     template.HTML
		prevHyphaName string
		nextHyphaName string
		hasSubhyphae  bool
		canDelete     = meta.U.CanProceed(path.Join("delete", h.CanonicalName()))
		canRename     = meta.U.CanProceed(path.Join("rename", h.CanonicalName()))
	)

	if cfg.ShowTree {
		subhyphae, prevHyphaName, nextHyphaName = tree.Tree(h)
		hasSubhyphae = len(subhyphae) > 0
	} else {
		prevHyphaName, nextHyphaName, hasSubhyphae = hyphae.Siblings(h)
	}

	data := map[string]any{
		"HyphaName":               h.CanonicalName(),
		"SubhyphaeHTML":           subhyphae,
		"PrevHyphaName":           prevHyphaName,
		"NextHyphaName":           nextHyphaName,
		"IsMyProfile":             isMyProfile,
		"ShowAdminPanel":          isMyProfile && meta.U.CanProceed("admin"),
		"NaviTitle":               hypview.NaviTitle(meta, h.CanonicalName()),
		"BacklinkCount":           hyphae.BacklinksCount(h.CanonicalName()),
		"GivenPermissionToModify": meta.U.CanProceed(path.Join("edit", h.CanonicalName())),
		"CanDelete":               canDelete,
		"CanRename":               canRename,
		"CanManageMedia":          meta.U.CanProceed(path.Join("media", h.CanonicalName())),
		"Categories":              cats,
		"CategoryNameOptions":     categories.ListOfCategories(),
		"IsMediaHypha":            false,
		"HasText":                 h.HasTextFile(),
		"HasSubhyphae":            hasSubhyphae,
	}
	slog.Info("reading hypha", "name", h.CanonicalName(), "can edit", data["GivenPermissionToModify"])
	meta.BodyAttributes = map[string]string{
		"cats": category_list,
	}

	switch h := h.(type) {
	case *hyphae.EmptyHypha:
		w.WriteHeader(http.StatusNotFound)
		data["Contents"] = ""
		data["CanDelete"] = canDelete && hasSubhyphae
		data["CanRename"] = canRename && hasSubhyphae
	case hyphae.ExistingHypha:
		fileContentsT, err := h.Text(history.FileReader())
		if err != nil {
			viewutil.HttpErr(meta, http.StatusInternalServerError, hyphaName, err.Error())
			return
		}

		ctx, _ := mycocontext.ContextFromStringInput(
			string(fileContentsT),
			mycoopts.MarkupOptions(hyphaName),
		)
		getOpenGraph, descVisitor, imgVisitor := tools.OpenGraphVisitors(ctx)
		ast := mycomarkup.BlockTree(ctx, descVisitor, imgVisitor)
		openGraph = template.HTML(getOpenGraph())
		contents = template.HTML(mycomarkup.BlocksToHTML(ctx, ast))

		if h, ok := h.(*hyphae.MediaHypha); ok {
			contents = template.HTML(mycoopts.Media(h, lc)) + contents
			data["IsMediaHypha"] = true
		}

		data["Contents"] = contents
		meta.HeadElements = append(meta.HeadElements, openGraph)

		// TODO: check head cats
		// TODO: check opengraph
	}
	_ = pageHypha.RenderTo(meta, data)
}

// handlerBacklinks lists all backlinks to a hypha.
func handlerBacklinks(w http.ResponseWriter, rq *http.Request) {
	hyphaName := util.HyphaNameFromRq(rq, "backlinks")

	_ = pageBacklinks.RenderTo(viewutil.MetaFrom(w, rq),
		map[string]any{
			"Addr":      cfg.Root + "backlinks/" + hyphaName,
			"HyphaName": hyphaName,
			"Backlinks": hyphae.BacklinksFor(hyphaName),
		})
}

func handlerSubhyphae(w http.ResponseWriter, rq *http.Request) {
	hyphaName := util.HyphaNameFromRq(rq, "subhyphae")
	h := hyphae.ByName(hyphaName)

	_ = pageSubhyphae.RenderTo(viewutil.MetaFrom(w, rq),
		map[string]any{
			"Addr":      cfg.Root + "subhyphae/" + hyphaName,
			"HyphaName": hyphaName,
			"Subhyphae": hyphae.Subhyphae(h),
		})
}

func handlerOrphans(w http.ResponseWriter, rq *http.Request) {
	_ = pageOrphans.RenderTo(viewutil.MetaFrom(w, rq),
		map[string]any{
			"Addr":    cfg.Root + "orphans",
			"Orphans": hyphae.Orphans(),
		})
}
