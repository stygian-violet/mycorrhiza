package search

import (
	"unicode/utf8"

	"github.com/bouncepaw/mycorrhiza/util"
)

type SearchResultLine = []string

type SearchResult struct {
	Hypha string
	Lines []SearchResultLine
}

type SearchResults struct {
	Hyphae []*SearchResult
	Complete bool
}

func NewSearchResults() *SearchResults {
	return &SearchResults{
		Hyphae: nil,
		Complete: true,
	}
}

func NewSearchResult(hypha string, line []string, maxLength int) *SearchResult {
	var lines []SearchResultLine = nil
	if line != nil {
		lines = []SearchResultLine{NewSearchResultLine(line, maxLength)}
	}
	return &SearchResult{
		Hypha: hypha,
		Lines: lines,
	}
}

func NewSearchResultLine(line []string, maxLength int) SearchResultLine {
	if maxLength == 0 {
		return nil
	}
	if maxLength < 0 || len(line) == 0 {
		return line
	}
	left, leftLength := 0, min(maxLength, utf8.RuneCountInString(line[0]))
	right, rightLength := 0, 0
	length := leftLength
L:
	for i := 1; i < len(line); i++ {
		l := utf8.RuneCountInString(line[i])
		switch {
		case length + l <= maxLength:
			right, rightLength = i, l
			length += l
		case i % 2 == 0:
			right, rightLength = i, maxLength - length
			break L
		case length - leftLength + l <= maxLength:
			right, rightLength = i, l
			leftLength = maxLength - (length - leftLength + l)
			length = maxLength
		default:
			break L
		}
	}
	var truncated bool
	line[left], truncated = util.TruncateLeft(line[left], leftLength)
	if truncated || left > 0 {
		line[left] = "…" + line[left]
	}
	if right > left {
		line[right], truncated = util.Truncate(line[right], rightLength)
		if truncated || right < len(line) - 1 {
			line[right] += "…"
		}
	}
	return line[left:right + 1]
}

func (sr *SearchResults) Empty() bool {
	return len(sr.Hyphae) == 0
}

func (sr *SearchResults) Last() *SearchResult {
	if sr.Empty() {
		return nil
	}
	return sr.Hyphae[len(sr.Hyphae) - 1]
}

func (sr *SearchResults) Append(
	hypha string,
	line []string,
	lineLength int,
	lineLimit uint,
) bool {
	last := sr.Last()
	if last == nil || last.Hypha != hypha {
		sr.Hyphae = append(sr.Hyphae, NewSearchResult(hypha, line, lineLength))
		return true
	}
	if lineLimit == 0 || uint(len(last.Lines)) < lineLimit {
		last.Append(NewSearchResultLine(line, lineLength))
		return true
	}
	return false
}

func (sr *SearchResults) Limit(limit int) bool {
	if limit >= 0 && len(sr.Hyphae) > limit {
		sr.Hyphae = sr.Hyphae[:limit]
		sr.Complete = false
		return false
	}
	return true
}

func (sr *SearchResult) Append(line []string) {
	sr.Lines = append(sr.Lines, line)
}
