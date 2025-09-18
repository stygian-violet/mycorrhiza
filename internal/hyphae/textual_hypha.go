package hyphae

import (
	"fmt"
	"os"

	"github.com/bouncepaw/mycorrhiza/util"
)

// TextualHypha is a hypha with text, and nothing else. An article, a note, a poem, whatnot.
type TextualHypha struct {
	canonicalName string
	mycoFilePath  string
}

func NewTextualHypha(canonicalName string) *TextualHypha {
	return &TextualHypha{
		canonicalName: canonicalName,
		mycoFilePath: TextFilePath(canonicalName),
	}
}

func (t *TextualHypha) CanonicalName() string {
	return t.canonicalName
}

func (t *TextualHypha) String() string {
	return fmt.Sprintf(
		"<textual hypha %s (%s)>",
		t.canonicalName, t.mycoFilePath,
	)
}

func (t *TextualHypha) Text(reader util.FileReader) (string, error) {
	text, err := reader.ReadFile(t.TextFilePath())
	switch {
	case os.IsNotExist(err):
		return "", nil
	case err != nil:
		return "", err
	default:
		return string(text), nil
	}
}

func (t *TextualHypha) FilePaths() []string {
	return []string{t.mycoFilePath}
}

func (t *TextualHypha) HasTextFile() bool {
	return true
}

func (t *TextualHypha) TextFilePath() string {
	return t.mycoFilePath
}

func (t *TextualHypha) WithName(name string) ExistingHypha {
	name = util.CanonicalName(name)
	return &TextualHypha{
		canonicalName: name,
		mycoFilePath: renameHyphaFile(t.mycoFilePath, t.canonicalName, name),
	}
}

func (t *TextualHypha) WithTextPath(mycoFilePath string) ExistingHypha {
	return &TextualHypha{
		canonicalName: t.CanonicalName(),
		mycoFilePath:  mycoFilePath,
	}
}

// WithMediaPath returns a new media hypha with the same name and text file as the given textual hypha. The new hypha is not stored yet.
func (t *TextualHypha) WithMediaPath(mediaFilePath string) ExistingHypha {
	return &MediaHypha{
		canonicalName: t.CanonicalName(),
		mycoFilePath:  t.TextFilePath(),
		mediaFilePath: mediaFilePath,
	}
}

func (t *TextualHypha) WithoutMedia() Hypha {
	return t
}
