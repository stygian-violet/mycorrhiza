package hyphae

import (
	"github.com/bouncepaw/mycorrhiza/util"
)

// Using set here seems like the most appropriate solution
type linkSet map[string]struct{}

func toLinkSet(xs []string) linkSet {
	result := make(linkSet)
	for _, x := range xs {
		result[x] = struct{}{}
	}
	return result
}

// backlinkIndexOperation is an operation for the backlink index. This operation is executed async-safe.
type backlinkIndexOperation interface {
	apply()
}

// backlinkIndexEdit contains data for backlink index update after a hypha edit
type backlinkIndexEdit struct {
	name     string
	oldLinks []string
	newLinks []string
}

// apply changes backlink index respective to the operation data
func (op backlinkIndexEdit) apply() {
	oldLinks := toLinkSet(op.oldLinks)
	newLinks := toLinkSet(op.newLinks)
	for link := range oldLinks {
		if _, exists := newLinks[link]; !exists {
			backlinksByName[link] = util.DeleteSorted(
				backlinksByName[link],
				util.PathographicCompare,
				op.name,
			)
		}
	}
	for link := range newLinks {
		if _, exists := oldLinks[link]; !exists {
			backlinksByName[link] = util.InsertSorted(
				backlinksByName[link],
				util.PathographicCompare,
				op.name,
			)
		}
	}
}

// backlinkIndexDeletion contains data for backlink index update after a hypha deletion
type backlinkIndexDeletion struct {
	name  string
	links []string
}

// apply changes backlink index respective to the operation data
func (op backlinkIndexDeletion) apply() {
	for _, link := range op.links {
		backlinksByName[link] = util.DeleteSorted(
			backlinksByName[link],
			util.PathographicCompare,
			op.name,
		)
	}
}

// backlinkIndexRenaming contains data for backlink index update after a hypha renaming
type backlinkIndexRenaming struct {
	oldName string
	newName string
	links   []string
}

// apply changes backlink index respective to the operation data
func (op backlinkIndexRenaming) apply() {
	for _, link := range op.links {
		backlinksByName[link] = util.ReplaceSorted(
			backlinksByName[link],
			util.PathographicCompare,
			op.oldName,
			op.newName,
		)
	}
}

// updateBacklinksAfterEdit is a creation/editing hook for backlinks index
func updateBacklinksAfterEdit(
	h Hypha, oldText string, newText string,
) backlinkIndexOperation {
	oldLinks := ExtractHyphaLinksFromString(h.CanonicalName(), oldText)
	newLinks := ExtractHyphaLinksFromString(h.CanonicalName(), newText)
	return backlinkIndexEdit{h.CanonicalName(), oldLinks, newLinks}
}

// updateBacklinksAfterDelete is a deletion hook for backlinks index
func updateBacklinksAfterDelete(
	h Hypha, oldText string,
) backlinkIndexOperation {
	oldLinks := ExtractHyphaLinksFromString(h.CanonicalName(), oldText)
	return backlinkIndexDeletion{h.CanonicalName(), oldLinks}
}

// updateBacklinksAfterRename is a renaming hook for backlinks index
func updateBacklinksAfterRename(
	h Hypha, oldName string, text string,
) backlinkIndexOperation {
	actualLinks := ExtractHyphaLinksFromString(h.CanonicalName(), text)
	return backlinkIndexRenaming{oldName, h.CanonicalName(), actualLinks}
}
