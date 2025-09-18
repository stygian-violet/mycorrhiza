package hyphae

import (
	"iter"
)

type renamePair struct {
	old ExistingHypha
	new ExistingHypha
}

type Op struct {
	done     bool
	remove   []ExistingHypha
	insert   []ExistingHypha
	rename   []renamePair
	backlink []backlinkIndexOperation
}

func IndexOperation() *Op {
	indexMutex.Lock()
	return &Op{}
}

func (op *Op) Exists(hyphaName string) bool {
	_, res := byNames[hyphaName]
	return res
}

func (op *Op) YieldSubhyphae(hypha ExistingHypha) iter.Seq[ExistingHypha] {
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
	return op
}

func (op *Op) WithHyphaRenamed(
	old ExistingHypha,
	new ExistingHypha,
	text string,
) *Op {
	if op.done {
		return op
	}
	op.rename = append(op.rename, renamePair{
		old: old,
		new: new,
	})
	op.backlink = append(
		op.backlink,
		updateBacklinksAfterRename(new, old.CanonicalName(), text),
	)
	return op
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
		op.WithHyphaRenamed(old, new, newText)
	} else {
		op.insert = append(op.insert, new)
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
		count += deleteHypha(hs.old)
		count += insertHypha(hs.new)
	}
	for _, h := range op.insert {
		count += insertHypha(h)
	}
	for _, b := range op.backlink {
		b.apply()
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
