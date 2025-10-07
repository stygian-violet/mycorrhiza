package hyphae

// File `iterators.go` contains stuff that iterates over hyphae.

import (
	"iter"
	"path"
	"slices"
	"strings"
)

// YieldExistingHyphae iterates over all hyphae and yields all existing ones.
func YieldExistingHyphae() iter.Seq[ExistingHypha] {
	return func(yield func (ExistingHypha) bool) {
		indexMutex.RLock()
		defer indexMutex.RUnlock()
		for _, h := range hyphae {
			if !yield(h) {
				break
			}
		}
	}
}

// YieldHyphaNamesContainingString picks hyphae with have a string in their title, sorts and iterates over them in alphabetical order.
func YieldHyphaNamesContainingString(query string) iter.Seq[string] {
	return func(yield func(string) bool) {
		indexMutex.RLock()
		defer indexMutex.RUnlock()
		for _, h := range hyphae {
			hyphaName := h.CanonicalName()
			if strings.Contains(hyphaName, query) && !yield(hyphaName) {
				return
			}
		}
	}
}

func yieldSubhyphae(h Hypha, lock bool) iter.Seq[ExistingHypha] {
	name := h.CanonicalName()
	prefix := name + "/"
	return func(yield func(ExistingHypha) bool) {
		if lock {
			indexMutex.RLock()
			defer indexMutex.RUnlock()
		}
		i, found := slices.BinarySearchFunc(hyphae, name, CompareName)
		if found {
			i++
		}
		n := len(hyphae)
		for i < n && strings.HasPrefix(hyphae[i].CanonicalName(), prefix) {
			if !yield(hyphae[i]) {
				return
			}
			i++
		}
	}
}

func YieldSubhyphae(h Hypha) iter.Seq[ExistingHypha] {
	return yieldSubhyphae(h, true)
}

func YieldSubhyphaeWithSiblings(
	h Hypha,
	prev *string,
	next *string,
) iter.Seq[ExistingHypha] {
	name := h.CanonicalName()
	parent := path.Dir(name)
	if parent == "/" || parent == "." {
		parent = ""
	} else {
		parent += "/"
	}
	return func(yield func(ExistingHypha) bool) {
		indexMutex.RLock()
		defer indexMutex.RUnlock()
		i, found := slices.BinarySearchFunc(hyphae, name, CompareName)
		*prev = childAtIndex(parent, i - 1)
		if found {
			i++
		}
		prefix := name + "/"
		yield_ := true
		for i < len(hyphae) && strings.HasPrefix(hyphae[i].CanonicalName(), prefix) {
			if yield_ && !yield(hyphae[i]) {
				yield_ = false
			}
			i++
		}
		*next = childAtIndex(parent, i)
	}
}
