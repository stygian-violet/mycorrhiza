package hyphae

// File `iterators.go` contains stuff that iterates over hyphae.

import (
	"iter"
	"sort"
	"strings"

	"github.com/bouncepaw/mycorrhiza/util"
)

// YieldExistingHyphae iterates over all hyphae and yields all existing ones.
func YieldExistingHyphae() iter.Seq[ExistingHypha] {
	return func(yield func (ExistingHypha) bool) {
		indexMutex.RLock()
		defer indexMutex.RUnlock()
		for _, h := range byNames {
			if !yield(h) {
				break
			}
		}
	}
}

func YieldExistingHyphaNames() iter.Seq[string] {
	return util.Map(func (h ExistingHypha) string {
		return h.CanonicalName()
	}, YieldExistingHyphae())
}

func yieldSubhyphae(h Hypha, lock bool) iter.Seq[ExistingHypha] {
	prefix := h.CanonicalName() + "/"
	return func(yield func(ExistingHypha) bool) {
		if lock {
			indexMutex.RLock()
			defer indexMutex.RUnlock()
		}
		for _, subh := range byNames {
			if strings.HasPrefix(subh.CanonicalName(), prefix) {
				if !yield(subh) {
					return
				}
			}
		}
	}
}

func YieldSubhyphae(h Hypha) iter.Seq[ExistingHypha] {
	return yieldSubhyphae(h, true)
}

// YieldHyphaBacklinks gets backlinks for the desired hypha, sorts and yields them one by one.
func YieldHyphaBacklinks(hyphaName string) iter.Seq[string] {
	hyphaName = util.CanonicalName(hyphaName)
	var out iter.Seq[string] = func(yield func(string) bool) {
		indexMutex.RLock()
		defer indexMutex.RUnlock()
		backlinks, exists := backlinksByName[hyphaName]
		if exists {
			for link := range backlinks {
				if !yield(link) {
					break
				}
			}
		}
	}
	sorted := PathographicSort(out)
	return sorted
}

// FilterHyphaeWithText filters the source channel and yields only those hyphae than have text parts.
func FilterHyphaeWithText(src iter.Seq[ExistingHypha]) iter.Seq[ExistingHypha] {
	return util.Filter(func (h ExistingHypha) bool {
		return h.HasTextFile()
	}, src)
}

// PathographicSort sorts paths inside the source channel, preserving the path tree structure
func PathographicSort(src iter.Seq[string]) iter.Seq[string] {
	// To make it unicode-friendly and lean, we cast every string into rune slices, sort, and only then cast them back
	raw := make([][]rune, 0)
	for h := range src {
		raw = append(raw, []rune(h))
	}
	sort.Slice(raw, func(i, j int) bool {
		const slash rune = 47 // == '/'
		// Classic lexicographical sort with a twist
		c := 0
		for {
			if c == len(raw[i]) {
				return true
			}
			if c == len(raw[j]) {
				return false
			}
			if raw[i][c] == raw[j][c] {
				c++
			} else {
				// The twist: subhyphae-awareness is about pushing slash upwards
				if raw[i][c] == slash {
					return true
				}
				if raw[j][c] == slash {
					return false
				}
				return raw[i][c] < raw[j][c]
			}
		}
	})
	return func(yield func(string) bool) {
		for _, name := range raw {
			if !yield(string(name)) {
				break
			}
		}
	}
}

// Subhyphae returns slice of subhyphae.
func Subhyphae(h Hypha) []ExistingHypha {
	var hyphae []ExistingHypha
	for subh := range YieldSubhyphae(h) {
		hyphae = append(hyphae, subh)
	}
	return hyphae
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
