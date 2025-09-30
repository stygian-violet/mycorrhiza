package hyphae

import (
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/interwiki"
	"github.com/bouncepaw/mycorrhiza/util"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"

	"git.sr.ht/~bouncepaw/mycomarkup/v5"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/blocks"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/links"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/mycocontext"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/options"
	"git.sr.ht/~bouncepaw/mycomarkup/v5/tools"
)

func ExtractionOptions(hyphaName string) options.Options {
	return options.Options{
		HyphaName:                hyphaName,
		TransclusionSupported:    true,
		InterwikiSupported:       true,
		LocalTargetCanonicalName: util.CanonicalName,
		LocalLinkHref: func(hyphaName string) string {
			return cfg.Root + "hypha/" + util.CanonicalName(hyphaName)
		},
		LocalImgSrc: func(hyphaName string) string {
			return cfg.Root + "binary/" + util.CanonicalName(hyphaName)
		},
		LinkHrefFormatForInterwikiPrefix: interwiki.HrefLinkFormatFor,
		ImgSrcFormatForInterwikiPrefix:   interwiki.ImgSrcFormatFor,
	}.FillTheRest()
}

func extractHyphaLinksFromContext(
	hyphaName string,
	ctx mycocontext.Context,
) []string {
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

// ExtractHyphaLinksFromContent extracts local hypha links from the provided text.
func ExtractHyphaLinksFromBytes(
	hyphaName string,
	contents []byte,
) []string {
	ctx, _ := mycocontext.ContextFromBytes(
		contents,
		ExtractionOptions(hyphaName),
	)
	return extractHyphaLinksFromContext(hyphaName, ctx)
}

// ExtractHyphaLinksFromStringContent extracts local hypha links from the provided text.
func ExtractHyphaLinksFromString(
	hyphaName string,
	contents string,
) []string {
	ctx, _ := mycocontext.ContextFromStringInput(
		contents,
		ExtractionOptions(hyphaName),
	)
	return extractHyphaLinksFromContext(hyphaName, ctx)
}

// ExtractHeaderLinksFromContent extracts all rocketlinks from the given text and returns them as header links.
func ExtractHeaderLinksFromString(
	hyphaName string,
	text string,
) []viewutil.HeaderLink {
	headerLinks := []viewutil.HeaderLink{}
	ctx, _ := mycocontext.ContextFromStringInput(text, ExtractionOptions(hyphaName))
	// We call for side-effects
	_ = mycomarkup.BlockTree(ctx, func(block blocks.Block) {
		switch launchpad := block.(type) {
		case blocks.LaunchPad:
			for _, rocket := range launchpad.Rockets {
				headerLinks = append(headerLinks, viewutil.NewHeaderLink(
					rocket.LinkHref(ctx),
					rocket.DisplayedText(),
				))
			}
		}
	})
	return headerLinks
}
