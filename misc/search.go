package misc

import (
	"errors"
	"strings"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/search"
)

var ErrTextSearchDisabled = errors.New("full text search is disabled")

func normalizeQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func fullTextSearch(query string, limit int) (*search.SearchResults, error) {
	if limit == 0 {
		return nil, ErrTextSearchDisabled
	}
	switch cfg.FullTextSearch {
	case cfg.FullTextGrep:
		return history.Grep(query, limit)
	default:
		return nil, ErrTextSearchDisabled
	}
}
