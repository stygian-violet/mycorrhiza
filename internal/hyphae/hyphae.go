package hyphae

import (
	"sync"

	"github.com/bouncepaw/mycorrhiza/util"
)

var (
	indexMutex sync.RWMutex

	byNames = make(map[string]ExistingHypha)
	backlinksByName = make(map[string]linkSet)
)

// ByName returns a hypha by name. It returns an *EmptyHypha if there is no such hypha. This function is the only source of empty hyphae.
func ByName(hyphaName string) (h Hypha) {
	indexMutex.RLock()
	h, recorded := byNames[hyphaName]
	indexMutex.RUnlock()
	if recorded {
		return h
	}
	return &EmptyHypha{
		canonicalName: hyphaName,
	}
}

func insertHypha(h ExistingHypha) int {
	_, exists := byNames[h.CanonicalName()]
	byNames[h.CanonicalName()] = h
	if !exists {
		return 1
	}
	return 0
}

func deleteHypha(h ExistingHypha) int {
	_, exists := byNames[h.CanonicalName()]
	if exists {
		delete(byNames, h.CanonicalName())
		return -1
	}
	return 0
}

// BacklinksCount returns the amount of backlinks to the hypha. Pass canonical names.
func BacklinksCount(hyphaName string) int {
	indexMutex.RLock()
	defer indexMutex.RUnlock()
	if links, exists := backlinksByName[hyphaName]; exists {
		return len(links)
	}
	return 0
}

func BacklinksFor(hyphaName string) []string {
	var backlinks []string
	for b := range YieldHyphaBacklinks(hyphaName) {
		backlinks = append(backlinks, b)
	}
	return backlinks
}

func Orphans() []string {
	var res []string
	names := YieldExistingHyphaNames()
	names = util.Filter(func (name string) bool {
		links, exists := backlinksByName[name]
		return !exists || len(links) == 0
	}, names)
	names = PathographicSort(names)
	for name := range names {
		res = append(res, name)
	}
	return res
}
