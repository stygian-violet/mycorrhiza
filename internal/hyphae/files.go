package hyphae

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/mimetype"
	"github.com/bouncepaw/mycorrhiza/internal/process"
)

type foundFile struct {
	hypha string
	path  string
	text  bool
}

// Index finds all hypha files in the full `path` and saves them to the hypha storage.
func Index(path string) {
	hop := history.ReadOperation()

	newByNames := make(map[string]ExistingHypha)
	newBacklinks := make(map[string]linkSet)
	newCount := 0
	ch := make(chan foundFile, 8)

	process.Go(func() {
		indexHelper(path, 0, ch)
		close(ch)
	})

	for file := range ch {
		var storedHypha Hypha
		storedHypha, exists := newByNames[file.hypha]
		if !exists {
			storedHypha = &EmptyHypha{canonicalName: file.hypha}
			newCount++
		} else {
			switch h := storedHypha.(type) {
			case *TextualHypha:
				if file.text {
					slog.Warn("File collision", "hypha", file.hypha,
						"usingFile", h.TextFilePath(), "insteadOf", file.path)
					continue
				}
			case *MediaHypha:
				if !file.text {
					slog.Warn("File collision", "hypha", file.hypha,
						"usingFile", h.MediaFilePath(), "insteadOf", file.path)
					continue
				}
			}
		}
		var updatedHypha ExistingHypha
		if file.text {
			updatedHypha = storedHypha.WithTextPath(file.path)
			err := indexBacklinks(hop, updatedHypha, newBacklinks)
			if err != nil {
				slog.Error("Failed to index backlinks", "hypha", storedHypha)
			}
		} else {
			updatedHypha = storedHypha.WithMediaPath(file.path)
		}
		newByNames[file.hypha] = updatedHypha
	}

	indexMutex.Lock()
	byNames = newByNames
	backlinksByName = newBacklinks
	setCount(newCount)
	indexMutex.Unlock()

	hop.Close()

	slog.Info("Indexed hyphae", "n", newCount)
}

func indexBacklinks(
	hop *history.ReadOp,
	h ExistingHypha,
	backlinks map[string]linkSet,
) error {
	text, err := h.Text(hop)
	if err != nil {
		return err
	}
	foundLinks := ExtractHyphaLinksFromContent(h.CanonicalName(), text)
	for _, link := range foundLinks {
		if _, exists := backlinks[link]; !exists {
			backlinks[link] = make(linkSet)
		}
		backlinks[link][h.CanonicalName()] = struct{}{}
	}
	return nil
}

// indexHelper finds all hypha files in the full `path` and sends them to the
// channel. Handling of duplicate entries and media and counting them is
// up to the caller.
func indexHelper(path string, nestLevel uint, ch chan foundFile) {
	nodes, err := os.ReadDir(path)
	if err != nil {
		slog.Error("Failed to read directory", "path", path, "err", err)
		process.Shutdown()
		return
	}

	for _, node := range nodes {
		// If this hypha looks like it can be a hypha path, go deeper. Do not
		// touch the .git folders for it has an administrative importance!
		if node.IsDir() && IsValidName(node.Name()) && node.Name() != ".git" {
			indexHelper(filepath.Join(path, node.Name()), nestLevel+1, ch)
			continue
		}

		var (
			hyphaPartPath           = filepath.Join(path, node.Name())
			hyphaName, isText, skip = mimetype.DataFromFilename(hyphaPartPath)
		)
		if !skip {
			file := foundFile {
				hypha: hyphaName,
				path: hyphaPartPath,
				text: isText,
			}
			select {
			case <-process.Done():
				return
			case ch <- file:
			}
		}
	}
}
