package mycoopts

import (
	"errors"
	"fmt"
	"html"
	"path/filepath"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/interwiki"
	"github.com/bouncepaw/mycorrhiza/l18n"
	"github.com/bouncepaw/mycorrhiza/util"

	"git.sr.ht/~bouncepaw/mycomarkup/v5/options"
)

func MarkupOptions(hyphaName string) options.Options {
	return options.Options{
		HyphaName:             hyphaName,
		WebSiteURL:            cfg.URL,
		TransclusionSupported: true,
		RedLinksSupported:     true,
		InterwikiSupported:    true,
		HyphaExists: func(hyphaName string) bool {
			switch hyphae.ByName(hyphaName).(type) {
			case *hyphae.EmptyHypha:
				return false
			default:
				return true
			}
		},
		IterateHyphaNamesWith: func(λ func(string)) {
			for h := range hyphae.YieldExistingHyphae() {
				λ(h.CanonicalName())
			}
		},
		HyphaHTMLData: func(hyphaName string) (rawText, binaryBlock string, err error) {
			switch h := hyphae.ByName(hyphaName).(type) {
			case *hyphae.EmptyHypha:
				err = errors.New("Hypha " + hyphaName + " does not exist")
			case *hyphae.TextualHypha:
				rawText, err = hyphae.FetchMycomarkupFile(h)
			case *hyphae.MediaHypha:
				rawText, err = hyphae.FetchMycomarkupFile(h)
				binaryBlock = mediaRaw(h)
			}
			return
		},
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

func mediaRaw(h *hyphae.MediaHypha) string {
	return Media(h, l18n.New("en", "en"))
}

func MediaFile(path string, url string, lc *l18n.Localizer) string {
	switch filepath.Ext(path) {
	case ".jpg", ".gif", ".png", ".webp", ".svg", ".ico":
		return fmt.Sprintf(
			`<div class="binary-container binary-container_with-img">
	<a href="%s"><img src="%s"/></a>
</div>`,
			url, url,
		)

	case ".ogg", ".webm", ".mp4":
		return fmt.Sprintf(
			`<div class="binary-container binary-container_with-video">
	<video controls>
		<source src="%s"/>
		<p>%s <a href="%s">%s</a></p>
	</video>
</div>`,
			url,
			html.EscapeString(lc.Get("ui.media_novideo")),
			url,
			html.EscapeString(lc.Get("ui.media_novideo_link")),
		)

	case ".mp3", ".wav", ".flac":
		return fmt.Sprintf(
			`<div class="binary-container binary-container_with-audio">
	<audio controls>
		<source src="%s"/>
		<p>%s <a href="%s">%s</a></p>
	</audio>
</div>`,
			url,
			html.EscapeString(lc.Get("ui.media_noaudio")),
			url,
			html.EscapeString(lc.Get("ui.media_noaudio_link")),
		)

	default:
		return fmt.Sprintf(
			`<div class="binary-container binary-container_with-nothing">
	<p><a href="%s">%s</a></p>
</div>`,
			url,
			html.EscapeString(lc.Get("ui.media_download")),
		)
	}
}

func Media(h *hyphae.MediaHypha, lc *l18n.Localizer) string {
	name := html.EscapeString(h.CanonicalName())
	path := h.MediaFilePath()
	url := fmt.Sprintf("%sbinary/%s", cfg.Root, name)
	return MediaFile(path, url, lc)
}
