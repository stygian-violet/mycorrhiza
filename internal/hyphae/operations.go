package hyphae

import (
	"iter"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

type opPart struct {
	hyphae    []ExistingHypha
	backlinks []backlinkIndexOperation
}

type Op struct {
	done        bool
	remove      opPart
	insert      opPart
	headerLinks []viewutil.HeaderLink
}

func IndexOperation() *Op {
	indexMutex.Lock()
	return &Op{}
}

func (op *Op) Exists(hyphaName string) bool {
	_, res := byNames[hyphaName]
	return res
}

func (op *Op) YieldSubhyphae(hypha Hypha) iter.Seq[ExistingHypha] {
	return yieldSubhyphae(hypha, false)
}

func (op *Op) WithHyphaCreated(h ExistingHypha, text string) *Op {
	if op.done {
		return op
	}
	op.insert.hyphae = append(op.insert.hyphae, h)
	if text != "" {
		op.insert.backlinks = append(
			op.insert.backlinks,
			updateBacklinksAfterEdit(h, "", text),
		)
	}
	if h.CanonicalName() == cfg.HeaderLinksHypha {
		op.headerLinks = ExtractHeaderLinksFromString(h.CanonicalName(), text)
	}
	return op
}

func (op *Op) WithHyphaDeleted(h ExistingHypha, text string) *Op {
	if op.done {
		return op
	}
	op.remove.hyphae = append(op.remove.hyphae, h)
	if text != "" {
		op.remove.backlinks = append(
			op.remove.backlinks,
			updateBacklinksAfterDelete(h, text),
		)
	}
	if h.CanonicalName() == cfg.HeaderLinksHypha && op.headerLinks == nil {
		op.headerLinks = viewutil.DefaultHeaderLinks()
	}
	return op
}

func (op *Op) WithHyphaRenamedPair(
	pair RenamingPair,
	text string,
) *Op {
	if op.done {
		return op
	}
	oldName := pair.From().CanonicalName()
	newName := pair.To().CanonicalName()
	op.remove.hyphae = append(op.remove.hyphae, pair.From())
	op.insert.hyphae = append(op.insert.hyphae, pair.To())
	op.insert.backlinks = append(
		op.insert.backlinks,
		updateBacklinksAfterRename(pair.To(), oldName, text),
	)
	switch {
	case newName == cfg.HeaderLinksHypha:
		op.headerLinks = ExtractHeaderLinksFromString(newName, text)
	case oldName == cfg.HeaderLinksHypha && op.headerLinks == nil:
		op.headerLinks = viewutil.DefaultHeaderLinks()
	}
	return op
}

func (op *Op) WithHyphaRenamed(
	old ExistingHypha,
	new ExistingHypha,
	text string,
) *Op {
	return op.WithHyphaRenamedPair(util.NewRenamingPair(old, new), text)
}

func (op *Op) WithHyphaTextChanged(
	old ExistingHypha, oldText string,
	new ExistingHypha, newText string,
) *Op {
	if op.done {
		return op
	}
	if oldText != newText {
		op.insert.backlinks = append(
			op.insert.backlinks,
			updateBacklinksAfterEdit(old, oldText, newText),
		)
	}
	if old.CanonicalName() != new.CanonicalName() {
		return op.WithHyphaRenamed(old, new, newText)
	}
	op.insert.hyphae = append(op.insert.hyphae, new)
	if new.CanonicalName() == cfg.HeaderLinksHypha {
		op.headerLinks = ExtractHeaderLinksFromString(
			new.CanonicalName(), newText,
		)
	}
	return op
}

func (op *Op) WithHyphaMediaChanged(old ExistingHypha, new ExistingHypha) *Op {
	if op.done {
		return op
	}
	op.insert.hyphae = append(op.insert.hyphae, new)
	return op
}

func (op *Op) Apply() *Op {
	if op.done {
		return op
	}
	count := modifyHyphae(op.remove.hyphae, op.insert.hyphae)
	for _, b := range op.remove.backlinks {
		b.apply()
	}
	for _, b := range op.insert.backlinks {
		b.apply()
	}
	if op.headerLinks != nil {
		viewutil.SetHeaderLinks(op.headerLinks)
	}
	addCount(count)
	indexMutex.Unlock()
	return op
}

func (op *Op) Abort() *Op {
	if op.done {
		return op
	}
	op.done = true
	indexMutex.Unlock()
	return op
}
