package interwiki

import (
	"embed"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"

	"github.com/gorilla/mux"
)

var (
	//go:embed *html
	fs            embed.FS
	ruTranslation = `
{{define "interwiki map"}}Интеркарта{{end}}
{{define "name"}}Название{{end}}
{{define "aliases"}}Псевдонимы{{end}}
{{define "aliases (,)"}}Псевдонимы (разделённые запятыми){{end}}
{{define "engine"}}Движок{{end}}
	{{define "engine/mycorrhiza"}}Микориза 🍄{{end}}
	{{define "engine/betula"}}Бетула 🌳{{end}}
	{{define "engine/agora"}}Агора ἀ{{end}}
	{{define "engine/generic"}}Любой сайт{{end}}
{{define "link href format"}}Строка форматирования атрибута href ссылки{{end}}
{{define "img src format"}}Строка форматирования атрибута src изображения{{end}}
{{define "unset map"}}Интеркарта не задана.{{end}}
{{define "documentation."}}Документация.{{end}}
{{define "edit separately."}}Изменяйте записи по отдельности.{{end}}
{{define "add interwiki entry"}}Добавить запись в интеркарту{{end}}
{{define "save"}}Сохранить{{end}}
{{define "delete"}}Удалить{{end}}
{{define "error"}}Ошибка{{end}}
`
	chainInterwiki viewutil.Chain
	chainModify    viewutil.Chain
)

func InitHandlers(rtr *mux.Router) {
	chainInterwiki = viewutil.CopyEnRuWith(fs, "view_interwiki.html", ruTranslation)
	chainModify    = viewutil.CopyEnRuWith(fs, "view_interwiki_modify.html", ruTranslation)
	rtr.HandleFunc("/interwiki", handlerInterwiki).Methods("GET")
	rtr.HandleFunc("/interwiki/add-entry", handlerAddEntry).Methods("GET", "POST")
	rtr.HandleFunc("/interwiki/modify-entry/{target}", handlerModifyEntry).Methods("GET", "POST")
}

type modifyData struct {
	*viewutil.BaseData
	*Wiki
	Action    string
	Error     string
	Name      string
	CanDelete bool
}

func handlerModifyEntry(w http.ResponseWriter, rq *http.Request) {
	var (
		name    = mux.Vars(rq)["target"]
		action  = rq.PostFormValue("action")
		oldWiki = ByName(name)
		newWiki = EmptyWiki()
		err     = error(nil)
	)
	if oldWiki.IsEmpty() {
		slog.Info(
			"Could not modify entry",
			"wiki", oldWiki, "action", action, "err", "does not exist",
		)
		viewutil.HandlerNotFound(w, rq)
		return
	}
	if rq.Method == "GET" {
		viewutil.ExecutePage(viewutil.MetaFrom(w, rq), chainModify, modifyData{
			BaseData:  &viewutil.BaseData{},
			Wiki:      oldWiki,
			Name:      util.BeautifulName(name),
			Action:    "modify-entry/" + name,
			CanDelete: true,
		})
		return
	}
	switch action {
	case "save":
		newWiki, err = FromRequest(rq)
	case "delete":
	default:
		err = fmt.Errorf("invalid action '%s'", action)
	}
	if err == nil {
		err = ReplaceEntry(oldWiki, newWiki)
	}
	if err != nil {
		slog.Info("Could not modify entry", "old", oldWiki, "new", newWiki, "err", err)
		wiki := oldWiki
		if !newWiki.IsEmpty() {
			wiki = newWiki
		}
		viewutil.ExecutePage(viewutil.MetaFrom(w, rq), chainModify, modifyData{
			BaseData:  &viewutil.BaseData{},
			Wiki:      wiki,
			Error:     err.Error(),
			Name:      util.BeautifulName(name),
			Action:    "modify-entry/" + name,
			CanDelete: true,
		})
		return
	}
	slog.Info("Modified entry", "old", oldWiki, "new", newWiki)
	http.Redirect(w, rq, cfg.Root + "interwiki", http.StatusSeeOther)
}

func handlerAddEntry(w http.ResponseWriter, rq *http.Request) {
	if rq.Method == "GET" {
		viewutil.ExecutePage(viewutil.MetaFrom(w, rq), chainModify, modifyData{
			BaseData: &viewutil.BaseData{},
			Wiki:     EmptyWiki(),
			Action:   "add-entry",
		})
		return
	}
	wiki, err := FromRequest(rq)
	if err == nil {
		err = AddEntry(wiki)
	}
	if err != nil {
		viewutil.ExecutePage(viewutil.MetaFrom(w, rq), chainModify, modifyData{
			BaseData:  &viewutil.BaseData{},
			Wiki:      wiki,
			Error:     err.Error(),
			Action:    "add-entry",
		})
	} else {
		http.Redirect(w, rq, cfg.Root + "interwiki", http.StatusSeeOther)
	}
}

type interwikiData struct {
	*viewutil.BaseData
	Entries []*Wiki
	CanEdit bool
	Error   string
}

func handlerInterwiki(w http.ResponseWriter, rq *http.Request) {
	meta := viewutil.MetaFrom(w, rq)
	canEdit := meta.U.CanProceed("interwiki/modify-entry")
	viewutil.ExecutePage(meta, chainInterwiki, interwikiData{
		BaseData: &viewutil.BaseData{},
		Entries:  Entries(),
		CanEdit:  canEdit,
		Error:    "",
	})
}
