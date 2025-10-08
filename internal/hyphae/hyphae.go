package hyphae

import (
	"math/rand"
	"strings"
	"sync"

	"github.com/bouncepaw/mycorrhiza/util"
)

var (
	indexMutex sync.RWMutex

	// TODO: use a different data structure?
	hyphae = []ExistingHypha(nil)
	byNames = make(map[string]ExistingHypha)
	backlinksByName = make(map[string][]string)
)

func modifyHyphae(remove []ExistingHypha, insert []ExistingHypha) {
	for _, h := range remove {
		delete(byNames, h.CanonicalName())
	}
	for _, h := range insert {
		byNames[h.CanonicalName()] = h
	}
	hyphae = util.ModifySorted(hyphae, Compare, remove, insert)
}

func childAtIndex(parent string, i int) string {
	if i < 0 || i >= len(hyphae) {
		return ""
	}
	name := hyphae[i].CanonicalName()
	if !strings.HasPrefix(name, parent) {
		return ""
	}
	child := name[len(parent):]
	j := strings.IndexByte(child, '/')
	if j < 0 {
		return name
	}
	return parent + child[:j]
}

// Count how many hyphae there are. This is a O(1), the number of hyphae is stored in memory.
func Count() (i int) {
	indexMutex.RLock()
	i = len(hyphae)
	indexMutex.RUnlock()
	return
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
	res := len(backlinksByName[hyphaName])
	indexMutex.RUnlock()
	return res
}

func BacklinksFor(hyphaName string) []string {
	res := []string(nil)
	hyphaName = util.CanonicalName(hyphaName)
	indexMutex.RLock()
	backlinks := backlinksByName[hyphaName]
	if len(backlinks) > 0 {
		res = make([]string, len(backlinks))
		copy(res, backlinks)
	}
	indexMutex.RUnlock()
	return res
}

func Orphans() []string {
	res := []string(nil)
	indexMutex.RLock()
	for _, h := range hyphae {
		name := h.CanonicalName()
		if len(backlinksByName[name]) == 0 {
			res = append(res, name)
		}
	}
	indexMutex.RUnlock()
	return res
}

// Subhyphae returns slice of subhyphae.
func Subhyphae(h Hypha) []ExistingHypha {
	var hyphae []ExistingHypha
	for subh := range YieldSubhyphae(h) {
		hyphae = append(hyphae, subh)
	}
	return hyphae
}

func HasSubhyphae(h Hypha) bool {
	for _ = range YieldSubhyphae(h) {
		return true
	}
	return false
}

func Siblings(h Hypha) (prev string, next string, hasSubhyphae bool) {
	for _ = range YieldSubhyphaeWithSiblings(h, &prev, &next) {
		hasSubhyphae = true
		break
	}
	return
}
