package hyphae

import (
	"math/rand"
	"slices"
	"sync"

	"github.com/bouncepaw/mycorrhiza/util"
)

var (
	indexMutex sync.RWMutex

	// TODO: use a different data structure?
	hyphae = []ExistingHypha(nil)
	byNames = make(map[string]ExistingHypha)
	backlinksByName = make(map[string]linkSet)
)

func modifyHyphae(remove []ExistingHypha, insert []ExistingHypha) int {
	for _, h := range remove {
		delete(byNames, h.CanonicalName())
	}
	for _, h := range insert {
		byNames[h.CanonicalName()] = h
	}
	count := len(hyphae)
	hyphae = util.ModifySorted(hyphae, Compare, remove, insert)
	count = len(hyphae) - count
	return count
}

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

func Random() ExistingHypha {
	indexMutex.RLock()
	defer indexMutex.RUnlock()
	n := len(hyphae)
	if n == 0 {
		return nil
	}
	return hyphae[rand.Intn(n)]
}

// AreFreeNames checks if all given `hyphaNames` are not taken. If they are not taken, `ok` is true. If not, `firstFailure` is the name of the first met hypha that is not free.
func AreFreeNames(hyphaNames ...string) (firstFailure string, ok bool) {
	indexMutex.RLock()
	defer indexMutex.RUnlock()
	for _, hn := range hyphaNames {
		if _, exists := byNames[hn]; exists {
			return hn, false
		}
	}
	return "", true
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
	res := []string(nil)
	hyphaName = util.CanonicalName(hyphaName)
	indexMutex.RLock()
	backlinks, exists := backlinksByName[hyphaName]
	if exists {
		res = make([]string, len(backlinks))
		i := 0
		for link := range backlinks {
			res[i] = link
			i++
		}
	}
	indexMutex.RUnlock()
	slices.SortFunc(res, util.PathographicCompare)
	return res
}

func Orphans() []string {
	res := []string(nil)
	indexMutex.RLock()
	for _, h := range hyphae {
		name := h.CanonicalName()
		links, exists := backlinksByName[name]
		if !exists || len(links) == 0 {
			res = append(res, name)
		}
	}
	indexMutex.RUnlock()
	return res
}
