package hyphae

import (
	"github.com/bouncepaw/mycorrhiza/util"

	"git.sr.ht/~bouncepaw/mycomarkup/v5"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/links"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/mycocontext"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/options"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/tools"
)

func extractionOptions(hyphaName string) options.Options {
	return options.Options{
		HyphaName:                hyphaName,
		TransclusionSupported:    true,
		LocalTargetCanonicalName: util.CanonicalName,
	}.FillTheRest()
}

// extractHyphaLinksFromContent extracts local hypha links from the provided text.
func extractHyphaLinksFromContent(hyphaName string, contents string) []string {
	ctx, _ := mycocontext.ContextFromStringInput(contents, extractionOptions(hyphaName))
	linkVisitor, getLinks := tools.LinkVisitor(ctx)
	// Ignore the result of BlockTree because we call it for linkVisitor.
	_ = mycomarkup.BlockTree(ctx, linkVisitor)
	foundLinks := getLinks()
	var result []string
	for _, link := range foundLinks {
		switch link := link.(type) {
		case *links.LocalLink:
			result = append(result, link.Target(ctx))
		}
	}
	return result
}
