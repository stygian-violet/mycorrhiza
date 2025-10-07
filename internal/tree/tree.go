package tree

import (
	"html/template"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/util"
)

func Tree(h hyphae.Hypha) (childrenHTML template.HTML, prev, next string) {
	tb := treeBuilder{parent: h.CanonicalName()}
	nodes := 0
	for h := range hyphae.YieldSubhyphaeWithSiblings(h, &prev, &next) {
		if cfg.MaxTreeNodes > 0 && nodes == cfg.MaxTreeNodes {
			tb.truncateAll(h.CanonicalName())
			break
		}
		tb.Append(h.CanonicalName())
		nodes++
	}
	tb.Close()
	childrenHTML = template.HTML(tb.String())
	return
}

type node struct {
	name      string
	hasList   bool
	truncated bool
}

type treeBuilder struct {
	buf    strings.Builder
	stack  []node
	parent string
}

func (tb *treeBuilder) Append(name string) {
	i := len(tb.parent) + 1
	level := 0
	for i < len(name) {
		if cfg.MaxTreeDepth > 0 && level == cfg.MaxTreeDepth {
			tb.truncate()
			return
		}
		j := strings.IndexByte(name[i:], '/')
		last := false
		if j < 0 {
			last = true
			j = len(name)
		} else {
			j += i
		}
		part := name[i:j]
		if level == len(tb.stack) || tb.stack[level].name != part {
			tb.push(level, last, name[i:j], name[:j])
		}
		i = j + 1
		level++
	}
}

func (tb *treeBuilder) Close() {
	tb.pop(len(tb.stack))
	tb.stack = nil
}

func (tb *treeBuilder) String() string {
	return tb.buf.String()
}

func (tb *treeBuilder) writeStrings(ss... string) {
	for _, s := range ss {
		tb.buf.WriteString(s)
	}
}

func (tb *treeBuilder) writeTruncation() {
	tb.buf.WriteString("<li class=\"subhyphae__truncated\">⋯</li>\n")
}

func (tb *treeBuilder) createList() {
	level := len(tb.stack) - 1
	if level >= 0 && !tb.stack[level].hasList {
		tb.stack[level].hasList = true
		tb.buf.WriteString("<ul>\n")
	}
}

func (tb *treeBuilder) truncate() {
	level := len(tb.stack) - 1
	if level >= 0 && !tb.stack[level].truncated {
		tb.stack[level].truncated = true
		tb.createList()
		tb.writeTruncation()
	}
}

func (tb *treeBuilder) truncateAll(next string) {
	level := 0
	for part := range strings.SplitSeq(next[len(tb.parent) + 1:], "/") {
		if level == len(tb.stack) || tb.stack[level].name != part {
			break
		}
		level++
	}
	tb.pop(len(tb.stack) - level)
	for len(tb.stack) > 0 {
		tb.truncate()
		tb.pop(1)
	}
	tb.writeTruncation()
}

func (tb *treeBuilder) push(level int, last bool, name string, path string) {
	if level < len(tb.stack) {
		tb.pop(len(tb.stack) - level)
	}
	tb.createList()
	tb.buf.WriteString("<li class=\"subhyphae__entry\">\n<a class=\"subhyphae__link")
	if !last {
		tb.buf.WriteString(" wikilink_new")
	}
	tb.writeStrings(
		"\" href=\"", cfg.Root, "hypha/", path, "\">",
		util.BeautifulName(name),"</a>\n",
	)
	tb.stack = append(tb.stack, node{name: name})
}

func (tb *treeBuilder) pop(count int) {
	i := len(tb.stack) - 1
	for j := 0; j < count; j++ {
		if tb.stack[i].hasList {
			tb.buf.WriteString("</ul>\n")
		}
		tb.buf.WriteString("</li>\n")
		i--
	}
	tb.stack = tb.stack[:i + 1]
}
