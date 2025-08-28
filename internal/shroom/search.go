package shroom

import (
	"iter"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/util"
)

// YieldHyphaNamesContainingString picks hyphae with have a string in their title, sorts and iterates over them in alphabetical order.
func YieldHyphaNamesContainingString(query string) iter.Seq[string] {
	all := hyphae.YieldExistingHyphaNames()
	filtered := util.Filter(func(name string) bool {
		return hyphaNameMatchesString(name, query)
	}, all)
	sorted := hyphae.PathographicSort(filtered)
	return sorted
}

// This thing gotta be changed one day, when a hero has time to implement a good searching algorithm.
func hyphaNameMatchesString(hyphaName, query string) bool {
	return strings.Contains(hyphaName, query)
}
