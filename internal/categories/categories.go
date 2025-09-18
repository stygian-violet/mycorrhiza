// Package categories provides category management.
//
// As per the long pondering, this is how categories (cats for short)
// work in Mycorrhiza:
//
//   - Cats are not hyphae. Cats are separate entities. This is not as
//     vibeful as I would have wanted, but seems to be more practical
//     due to //the reasons//.
//   - Cats are stored outside of git. Instead, they are stored in a
//     JSON file, path to which is determined by files.CategoriesJSON.
//   - Due to not being stored in git, no cat history is tracked, and
//     cat operations are not mentioned on the recent changes page.
//   - For cat A, if there are 0 hyphae in the cat, cat A does not
//     exist. If there are 1 or more hyphae in the cat, cat A exists.
//
// List of things to do with categories later:
//
//   - Forbid / in cat names.
//   - Rename categories.
//   - Delete categories.
//   - Bind hyphae.
package categories

import (
	"sync"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/process"
	"github.com/bouncepaw/mycorrhiza/util"
)

// ListOfCategories returns unsorted names of all categories.
func ListOfCategories() (categoryList []string) {
	mutex.RLock()
	for cat, _ := range categoryToHyphae {
		categoryList = append(categoryList, cat)
	}
	mutex.RUnlock()
	return categoryList
}

// categoriesWithHypha returns what categories have the given hypha. The hypha name must be canonical.
func categoriesWithHypha(hyphaName string) (categoryList []string) {
	if node, ok := hyphaToCategories[hyphaName]; ok {
		return node.categoryList
	} else {
		return nil
	}
}

// CategoriesWithHypha returns what categories have the given hypha. The hypha name must be canonical.
func CategoriesWithHypha(hyphaName string) (categoryList []string) {
	mutex.RLock()
	res := categoriesWithHypha(hyphaName)
	mutex.RUnlock()
	return res
}

// HyphaeInCategory returns what hyphae are in the category. If the returned slice is empty, the category does not exist, and vice versa. The category name must be canonical.
func HyphaeInCategory(catName string) (hyphaList []string) {
	mutex.RLock()
	defer mutex.RUnlock()
	if node, ok := categoryToHyphae[catName]; ok {
		return node.hyphaList
	} else {
		return nil
	}
}

var mutex sync.RWMutex

// addHyphaeToCategory adds the hypha to the category and updates the records on the disk. If the hypha is already in the category, nothing happens. Pass canonical names.
func addHyphaToCategory(catName string, hyphaName string) {
	if node, ok := hyphaToCategories[hyphaName]; ok {
		node.storeCategory(catName)
	} else {
		hyphaToCategories[hyphaName] = &hyphaNode{categoryList: []string{catName}}
	}

	if node, ok := categoryToHyphae[catName]; ok {
		node.storeHypha(hyphaName)
	} else {
		categoryToHyphae[catName] = &categoryNode{hyphaList: []string{hyphaName}}
	}
}

// AddHyphaeToCategory adds the hyphae to the category and updates the records on the disk. If a hypha is already in the category, nothing happens. Pass canonical names.
func AddHyphaeToCategory(catName string, hyphaNames... string) {
	mutex.Lock()
	for _, hyphaName := range hyphaNames {
		addHyphaToCategory(catName, hyphaName)
	}
	mutex.Unlock()
	process.Go(saveToDisk)
}

// RemoveHyphaeFromCategory removes the hyphae from the category and updates the records on the disk. If a hypha is not in the category, nothing happens. Pass canonical names.
func RemoveHyphaeFromCategory(catName string, hyphaNames... string) {
	mutex.Lock()
	for _, hyphaName := range hyphaNames {
		if node, ok := hyphaToCategories[hyphaName]; ok {
			node.removeCategory(catName)
			if len(node.categoryList) == 0 {
				delete(hyphaToCategories, hyphaName)
			}
		}

		if node, ok := categoryToHyphae[catName]; ok {
			node.removeHypha(hyphaName)
			if len(node.hyphaList) == 0 {
				delete(categoryToHyphae, catName)
			}
		}
	}
	mutex.Unlock()
	process.Go(saveToDisk)
}

// RemoveHyphaeFromAllCategories removes the given hyphae from all the categories.
func RemoveHyphaeFromAllCategories(hyphaNames... string) {
	mutex.Lock()
	for _, hyphaName := range hyphaNames {
		cats := categoriesWithHypha(hyphaName)
		for _, cat := range cats {
			if node, ok := hyphaToCategories[hyphaName]; ok {
				node.removeCategory(cat)
				if len(node.categoryList) == 0 {
					delete(hyphaToCategories, hyphaName)
				}
			}

			if node, ok := categoryToHyphae[cat]; ok {
				node.removeHypha(hyphaName)
				if len(node.hyphaList) == 0 {
					delete(categoryToHyphae, cat)
				}
			}
		}
	}
	mutex.Unlock()
	process.Go(saveToDisk)
}

// RenameHyphaeInAllCategories finds all mentions of oldName and replaces them with newName. Pass canonical names. Make sure newName is not taken. If oldName is not in any category, RenameHyphaeInAllCategories is a no-op.
func RenameHyphaeInAllCategories(
	leftRedirections bool,
	pairs... util.RenamingPair[string],
) {
	mutex.Lock()
	for _, pair := range pairs {
		oldName := pair.From()
		newName := pair.To()
		if node, ok := hyphaToCategories[oldName]; ok {
			hyphaToCategories[newName] = node
			delete(hyphaToCategories, oldName) // node should still be in memory 🙏
			for _, catName := range node.categoryList {
				if catNode, ok := categoryToHyphae[catName]; ok {
					catNode.removeHypha(oldName)
					catNode.storeHypha(newName)
				}
			}
		}
		if leftRedirections {
			addHyphaToCategory(cfg.RedirectionCategory, oldName)
		}
	}
	mutex.Unlock()
	process.Go(saveToDisk)
}
