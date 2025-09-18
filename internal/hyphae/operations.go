package hyphae

import (
	"iter"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

type Op struct {
	done        bool
	remove      []ExistingHypha
	insert      []ExistingHypha
	rename      []RenamingPair
	backlink    []backlinkIndexOperation
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
	op.insert = append(op.insert, h)
	if text != "" {
		op.backlink = append(
			op.backlink,
			updateBacklinksAfterEdit(h, "", text),
		)
	}
	if h.CanonicalName() == cfg.HeaderLinksHypha {
		op.headerLinks = ExtractHeaderLinksFromContent(h.CanonicalName(), text)
	}
	return op
}

func (op *Op) WithHyphaDeleted(h ExistingHypha, text string) *Op {
	if op.done {
		return op
	}
	op.remove = append(op.remove, h)
	if text != "" {
		op.backlink = append(
			op.backlink,
			updateBacklinksAfterDelete(h, text),
		)
	}
	if h.CanonicalName() == cfg.HeaderLinksHypha {
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
	op.rename = append(op.rename, pair)
	op.backlink = append(
		op.backlink,
		updateBacklinksAfterRename(pair.To(), oldName, text),
	)
	switch {
	case newName == cfg.HeaderLinksHypha:
		op.headerLinks = ExtractHeaderLinksFromContent(newName, text)
	case oldName == cfg.HeaderLinksHypha:
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
		op.backlink = append(
			op.backlink,
			updateBacklinksAfterEdit(old, oldText, newText),
		)
	}
	if old.CanonicalName() != new.CanonicalName() {
		return op.WithHyphaRenamed(old, new, newText)
	}
	op.insert = append(op.insert, new)
	if new.CanonicalName() == cfg.HeaderLinksHypha {
		op.headerLinks = ExtractHeaderLinksFromContent(
			new.CanonicalName(), newText,
		)
	}
	return op
}

func (op *Op) WithHyphaMediaChanged(old ExistingHypha, new ExistingHypha) *Op {
	if op.done {
		return op
	}
	op.insert = append(op.insert, new)
	return op
}

func (op *Op) Apply() *Op {
	if op.done {
		return op
	}
	count := 0
	for _, h := range op.remove {
		count += deleteHypha(h)
	}
	for _, hs := range op.rename {
		count += deleteHypha(hs.From())
		count += insertHypha(hs.To())
	}
	for _, h := range op.insert {
		count += insertHypha(h)
	}
	for _, b := range op.backlink {
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
