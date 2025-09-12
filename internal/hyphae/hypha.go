// Package hyphae manages hypha storage and hypha types.
package hyphae

import (
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/files"
)

// hyphaNamePattern is a pattern which all hyphae names must match.
var hyphaNamePattern = regexp.MustCompile(`^[^?!:#@><*|"'&%{}]+$`)

// IsValidName checks for invalid characters and path traversals.
func IsValidName(hyphaName string) bool {
	if !hyphaNamePattern.MatchString(hyphaName) {
		return false
	}
	for _, segment := range strings.Split(hyphaName, "/") {
		if segment == ".git" || segment == ".." {
			return false
		}
	}
	return true
}

// Hypha is the hypha you know and love.
type Hypha interface {
	sync.Locker

	// CanonicalName returns the canonical name of the hypha.
	//
	//     util.CanonicalName(h.CanonicalName()) == h.CanonicalName()
	CanonicalName() string

	FilePaths() []string
}

// ByName returns a hypha by name. It returns an *EmptyHypha if there is no such hypha. This function is the only source of empty hyphae.
func ByName(hyphaName string) (h Hypha) {
	byNamesMutex.Lock()
	defer byNamesMutex.Unlock()
	h, recorded := byNames[hyphaName]
	if recorded {
		return h
	}
	return &EmptyHypha{
		canonicalName: hyphaName,
	}
}

func AtRevision(hyphaName string, revHash string) (Hypha, error) {
	text, media, err := history.HyphaFilesAtRevision(hyphaName, revHash)
	if text != "" {
		text = filepath.Join(files.HyphaeDir(), text)
	}
	if media != "" {
		media = filepath.Join(files.HyphaeDir(), media)
	}
	switch {
	case err != nil:
		return &EmptyHypha{
			canonicalName: hyphaName,
		}, err
	case text == "" && media == "":
		return &EmptyHypha{
			canonicalName: hyphaName,
		}, nil
	case media == "":
		return &TextualHypha{
			canonicalName: hyphaName,
			mycoFilePath: text,
		}, nil
	default:
		return &MediaHypha{
			canonicalName: hyphaName,
			mycoFilePath: text,
			mediaFilePath: media,
		}, nil
	}
}
