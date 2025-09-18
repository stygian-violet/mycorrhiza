package hyphae

import (
	"fmt"
	"os"

	"github.com/bouncepaw/mycorrhiza/util"
)

type MediaHypha struct {
	canonicalName string
	mycoFilePath  string
	mediaFilePath string
}

func (m *MediaHypha) CanonicalName() string {
	return m.canonicalName
}

func (m *MediaHypha) String() string {
	return fmt.Sprintf(
		"<media hypha %s (text=%s media=%s)>",
		m.canonicalName, m.mycoFilePath, m.mediaFilePath,
	)
}

func (m *MediaHypha) Text(reader util.FileReader) (string, error) {
	if !m.HasTextFile() {
		return "", nil
	}
	text, err := reader.ReadFile(m.TextFilePath())
	switch {
	case os.IsNotExist(err):
		return "", nil
	case err != nil:
		return "", err
	default:
		return string(text), nil
	}
}

func (m *MediaHypha) FilePaths() []string {
	if !m.HasTextFile() {
		return []string{m.mediaFilePath}
	}
	return []string{m.mycoFilePath, m.mediaFilePath}
}

func (m *MediaHypha) TextFilePath() string {
	if m.mycoFilePath == "" {
		return TextFilePath(m.canonicalName)
	}
	return m.mycoFilePath
}

func (m *MediaHypha) HasTextFile() bool {
	return m.mycoFilePath != ""
}

func (m *MediaHypha) MediaFilePath() string {
	return m.mediaFilePath
}

func (m *MediaHypha) WithName(name string) ExistingHypha {
	name = util.CanonicalName(name)
	return &MediaHypha{
		canonicalName: name,
		mycoFilePath: renameHyphaFile(m.mycoFilePath, m.canonicalName, name),
		mediaFilePath: renameHyphaFile(m.mediaFilePath, m.canonicalName, name),
	}
}

func (m *MediaHypha) WithTextPath(mycoFilePath string) ExistingHypha {
	return &MediaHypha{
		canonicalName: m.CanonicalName(),
		mycoFilePath:  mycoFilePath,
		mediaFilePath: m.MediaFilePath(),
	}
}

func (m *MediaHypha) WithMediaPath(mediaFilePath string) ExistingHypha {
	return &MediaHypha{
		canonicalName: m.CanonicalName(),
		mycoFilePath:  m.TextFilePath(),
		mediaFilePath: mediaFilePath,
	}
}

func (m *MediaHypha) WithoutMedia() Hypha {
	if m.HasTextFile() {
		return &TextualHypha{
			canonicalName: m.CanonicalName(),
			mycoFilePath:  m.TextFilePath(),
		}
	}
	return &EmptyHypha{
		canonicalName: m.CanonicalName(),
	}
}
