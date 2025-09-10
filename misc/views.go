package misc

import (
	"embed"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/search"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

var (
	//go:embed *html
	fs                          embed.FS
	chainList, chainTitleSearch viewutil.Chain
	chainTextSearch             viewutil.Chain
	ruTranslation               = `
{{define "list of hyphae"}}Список гиф{{end}}
{{define "search:"}}Поиск: {{.}}{{end}}
{{define "search results for"}}Результаты поиска для «{{.}}»{{end}}
{{define "search no results"}}Ничего не найдено.{{end}}
{{define "x total"}}{{.}} всего.{{end}}
{{define "go to hypha"}}Перейти к гифе <a class="wikilink{{if .HasExactMatch | not}} wikilink_new{{end}}" href="{{.Meta.Root}}hypha/{{.MatchedHyphaName}}">{{beautifulName .MatchedHyphaName}}</a>.{{end}}
{{define "title search results"}}В названии{{end}}
{{define "text search results"}}В тексте{{end}}
{{define "search more"}}Больше результатов поиска{{end}}
{{define "search in text"}}Поиск в тексте{{end}}
{{define "search not complete"}}{{end}}
`
)

func initViews() {
	chainList = viewutil.CopyEnRuWith(fs, "view_list.html", ruTranslation)
	chainTitleSearch = viewutil.CopyEnRuWith(fs, "view_title_search.html", ruTranslation)
	chainTextSearch = viewutil.CopyEnRuWith(fs, "view_text_search.html", ruTranslation)
}

type listDatum struct {
	Name string
	Ext  string
}

type listData struct {
	*viewutil.BaseData
	Entries    []listDatum
	HyphaCount int
}

func viewList(meta viewutil.Meta, entries []listDatum) {
	viewutil.ExecutePage(meta, chainList, listData{
		BaseData:   &viewutil.BaseData{},
		Entries:    entries,
		HyphaCount: hyphae.Count(),
	})
}

type titleSearchData struct {
	*viewutil.BaseData
	Query             string
	Results           []string
	TextResults       *search.SearchResults
	MatchedHyphaName  string
	HasExactMatch     bool
	HasTextResults    bool
	HasAnyResults     bool
	HasTextSearchLink bool
}

func viewTitleSearch(meta viewutil.Meta, query string, hyphaName string, hasExactMatch bool, results []string, textResults *search.SearchResults) {
	hasTextResults := textResults != nil && len(textResults.Hyphae) > 0
	hasTextSearchLink := cfg.FullTextSearchPage &&
		(cfg.FullTextLowerLimit == 0 ||
			textResults == nil ||
			!textResults.Complete) &&
		meta.U.CanProceed("text-search")
	viewutil.ExecutePage(meta, chainTitleSearch, titleSearchData{
		BaseData:          &viewutil.BaseData{},
		Query:             query,
		Results:           results,
		TextResults:       textResults,
		MatchedHyphaName:  hyphaName,
		HasExactMatch:     hasExactMatch,
		HasTextResults:    hasTextResults,
		HasAnyResults:     hasTextResults || len(results) > 0,
		HasTextSearchLink: hasTextSearchLink,
	})
}


type textSearchData struct {
	*viewutil.BaseData
	Query            string
	Results          *search.SearchResults
	HasResults       bool
}

func viewTextSearch(meta viewutil.Meta, query string, results *search.SearchResults) {
	viewutil.ExecutePage(meta, chainTextSearch, textSearchData{
		BaseData:         &viewutil.BaseData{},
		Query:            query,
		Results:          results,
		HasResults:       results != nil,
	})
}
