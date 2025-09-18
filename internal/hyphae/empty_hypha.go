package hyphae

import (
	"fmt"

	"github.com/bouncepaw/mycorrhiza/util"
)

// EmptyHypha is a hypha that does not exist and is not stored anywhere. You get one when querying for a hypha that was not created before.
type EmptyHypha struct {
	canonicalName string
}

func NewEmptyHypha(canonicalName string) *EmptyHypha {
	return &EmptyHypha{
		canonicalName: canonicalName,
	}
}

func (e *EmptyHypha) CanonicalName() string {
	return e.canonicalName
}

func (e *EmptyHypha) String() string {
	return fmt.Sprintf("<empty hypha %s>", e.canonicalName)
}

func (e *EmptyHypha) Text(reader util.FileReader) (string, error) {
	return "", nil
}

func (e *EmptyHypha) HasTextFile() bool {
	return false
}

func (e *EmptyHypha) TextFilePath() string {
	return TextFilePath(e.canonicalName)
}

func (e *EmptyHypha) FilePaths() []string {
	return nil
}

// WithTextPath returns a new textual hypha with the same name as the given empty hypha. The created hypha is not stored yet.
func (e *EmptyHypha) WithTextPath(mycoFilePath string) ExistingHypha {
	return &TextualHypha{
		canonicalName: e.CanonicalName(),
		mycoFilePath:  mycoFilePath,
	}
}

// WithMediaPath returns a new media hypha with the same name as the given empty hypha. The created hypha is not stored yet.
func (e *EmptyHypha) WithMediaPath(mediaFilePath string) ExistingHypha {
	return &MediaHypha{
		canonicalName: e.CanonicalName(),
		mycoFilePath:  "",
		mediaFilePath: mediaFilePath,
	}
}

func (e *EmptyHypha) WithoutMedia() Hypha {
	return e
}
