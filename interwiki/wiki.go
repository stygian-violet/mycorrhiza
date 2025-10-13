package interwiki

import (
	"encoding/json"
	"fmt"
	"iter"
	"net/http"
	"slices"
	"strings"

	"github.com/bouncepaw/mycorrhiza/util"
)

// WikiEngine is an enumeration of supported interwiki targets.
type WikiEngine string

const (
	Mycorrhiza WikiEngine = "mycorrhiza"
	Betula     WikiEngine = "betula"
	Agora      WikiEngine = "agora"
	// Generic is any website.
	Generic    WikiEngine = "generic"
)

func (we WikiEngine) Valid() bool {
	switch we {
	case Mycorrhiza, Betula, Agora, Generic:
		return true
	}
	return false
}

func (we WikiEngine) LinkHrefFormat() string {
	switch we {
	case Mycorrhiza:
		return "%s/hypha/{NAME}"
	case Betula:
		return "%s/{BETULA-NAME}"
	case Agora:
		return "%s/node/{NAME}"
	default:
		return "%s/{NAME}"
	}
}

func (we WikiEngine) ImgSrcFormat() string {
	switch we {
	case Mycorrhiza:
		return "%s/binary/{NAME}"
	default:
		return "%s/{NAME}"
	}
}

// Wiki is an entry in the interwiki map.
type Wiki struct {
	// Name is the name of the wiki, and is also one of the possible prefices.
	name string

	// Aliases are alternative prefices you can use instead of Name. This slice can be empty.
	aliases []string

	// URL is the address of the wiki.
	url string

	// LinkHrefFormat is a format string for interwiki links. See Mycomarkup internal docs hidden deep inside for more information.
	//
	// This field is optional. If it is not set, it is derived from other data. See the code.
	linkHrefFormat string

	imgSrcFormat string

	// Engine is the engine of the wiki. Invalid values will result in a start-up error.
	engine WikiEngine
}

var emptyWiki = &Wiki{}

// Wiki is an entry in the interwiki map.
type wikiJson struct {
	Name           string     `json:"name"`
	Aliases        []string   `json:"aliases,omitempty"`
	URL            string     `json:"url"`
	LinkHrefFormat string     `json:"link_href_format"`
	ImgSrcFormat   string     `json:"img_src_format"`
	Engine         WikiEngine `json:"engine"`
}

func EmptyWiki() *Wiki {
	return emptyWiki
}

func FromRequest(rq *http.Request) (*Wiki, error) {
	wiki := &Wiki{
		name:           rq.PostFormValue("name"),
		aliases:        strings.Split(rq.PostFormValue("aliases"), ","),
		url:            rq.PostFormValue("url"),
		linkHrefFormat: rq.PostFormValue("link-href-format"),
		imgSrcFormat:   rq.PostFormValue("img-src-format"),
		engine:         WikiEngine(rq.PostFormValue("engine")),
	}
	if err := wiki.canonize(); err != nil {
		return EmptyWiki(), err
	}
	return wiki, nil
}

func Compare(w *Wiki, x *Wiki) int {
	return strings.Compare(w.name, x.name)
}

func (w *Wiki) IsEmpty() bool {
	return w == nil || w == emptyWiki
}

func (w *Wiki) Name() string {
	return w.name
}

func (w *Wiki) Names() iter.Seq[string] {
	return func(yield func(string) bool) {
		if w.IsEmpty() || !yield(w.name) {
			return
		}
		for _, name := range w.aliases {
			if !yield(name) {
				return
			}
		}
	}
}

func (w *Wiki) URL() string {
	return w.url
}

func (w *Wiki) Aliases() iter.Seq2[int, string] {
	return slices.All(w.aliases)
}

func (w *Wiki) LinkHrefFormat() string {
	return w.linkHrefFormat
}

func (w *Wiki) ImgSrcFormat() string {
	return w.imgSrcFormat
}

func (w *Wiki) Engine() WikiEngine {
	return w.engine
}

func (w *Wiki) String() string {
	switch {
	case w == nil:
		return "<nil wiki>"
	case w.IsEmpty():
		return "<empty wiki>"
	default:
		return fmt.Sprintf("<%s wiki '%s' (url %s)>", w.engine, w.name, w.url)
	}
}

func (w *Wiki) MarshalJSON() ([]byte, error) {
	return json.Marshal(wikiJson{
		Name:           w.name,
		Aliases:        w.aliases,
		URL:            w.url,
		LinkHrefFormat: w.linkHrefFormat,
		ImgSrcFormat:   w.imgSrcFormat,
		Engine:         w.engine,
	})
}

func (w *Wiki) UnmarshalJSON(b []byte) error {
	var data wikiJson
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	w.name = data.Name
	w.aliases = data.Aliases
	w.url = data.URL
	w.linkHrefFormat = data.LinkHrefFormat
	w.imgSrcFormat = data.ImgSrcFormat
	w.engine = data.Engine
	return w.canonize()
}

func (w *Wiki) canonize() error {
	w.name = util.CanonicalName(w.name)
	w.url = strings.TrimSpace(w.url)

	switch {
	case w.name == "":
		return fmt.Errorf("missing wiki name: %s", w)
	case w.url == "":
		return fmt.Errorf("missing wiki url: %s", w)
	case !w.engine.Valid():
		return fmt.Errorf("invalid wiki engine: %s: %s", w.engine, w)
	}

	for i, alias := range w.aliases {
		w.aliases[i] = util.CanonicalName(alias)
	}
	nameUsed := map[string]bool { w.name: true }
	w.aliases = slices.DeleteFunc(w.aliases, func(name string) bool {
		if name == "" || nameUsed[name] {
			return true
		}
		nameUsed[name] = true
		return false
	})

	if w.linkHrefFormat == "" || w.engine != Generic {
		w.linkHrefFormat = fmt.Sprintf(w.engine.LinkHrefFormat(), w.url)
	}

	if w.imgSrcFormat == "" || w.engine != Generic {
		w.imgSrcFormat = fmt.Sprintf(w.engine.ImgSrcFormat(), w.url)
	}

	return nil
}
