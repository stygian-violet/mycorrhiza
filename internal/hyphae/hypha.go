// Package hyphae manages hypha storage and hypha types.
package hyphae

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/util"
)

// Hypha is the hypha you know and love.
type Hypha interface {
	fmt.Stringer

	// CanonicalName returns the canonical name of the hypha.
	//
	//     util.CanonicalName(h.CanonicalName()) == h.CanonicalName()
	CanonicalName() string

	Text(reader util.FileReader) (string, error)

	HasTextFile() bool
	TextFilePath() string
	FilePaths() []string

	WithTextPath(path string) ExistingHypha
	WithMediaPath(path string) ExistingHypha
	WithoutMedia() Hypha
}

// ExistingHypha is not EmptyHypha. *MediaHypha and *TextualHypha implement this interface.
type ExistingHypha interface {
	Hypha

	WithName(name string) ExistingHypha
}

type RenamingPair = util.RenamingPair[ExistingHypha]

// hyphaNamePattern is a pattern which all hyphae names must match.
var hyphaNamePattern = regexp.MustCompile(`^[^?!:#@><*|"'&%{}]+$`)

func renameHyphaFile(path string, oldHyphaName string, newHyphaName string) string {
	namepart := util.CanonicalName(filepath.ToSlash(util.ShorterPath(path)))
	namepart = strings.Replace(namepart, oldHyphaName, newHyphaName, 1)
	return filepath.Join(files.HyphaeDir(), filepath.FromSlash(namepart))
}

func FilePath(hyphaName string) string {
	return filepath.Join(files.HyphaeDir(), filepath.FromSlash(hyphaName))
}

func TextFilePath(hyphaName string) string {
	return FilePath(hyphaName) + ".myco"
}

func Compare(h ExistingHypha, g ExistingHypha) int {
	return util.PathographicCompare(h.CanonicalName(), g.CanonicalName())
}

func CompareName(h ExistingHypha, name string) int {
	return util.PathographicCompare(h.CanonicalName(), name)
}

// IsValidName checks for invalid characters and path traversals.
func IsValidName(hyphaName string) bool {
	if !hyphaNamePattern.MatchString(hyphaName) {
		return false
	}
	for _, segment := range strings.Split(hyphaName, "/") {
		if segment == ".git" || segment == ".." || segment == "." {
			return false
		}
	}
	return true
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
