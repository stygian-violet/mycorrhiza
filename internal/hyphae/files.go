package hyphae

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/mimetype"
	"github.com/bouncepaw/mycorrhiza/internal/process"
)

type foundFile struct {
	hypha  string
	path   string
	text   []byte
	isText bool
}

// Index finds all hypha files in the full `path` and saves them to the hypha storage.
func Index(path string) error {
	newByNames := make(map[string]ExistingHypha)
	newBacklinks := make(map[string]linkSet)
	newCount := 0
	ch := make(chan foundFile, 8)
	err := error(nil)

	process.Go(func() {
		hop := history.ReadOperation()
		err = indexHelper(hop, process.Context(), ch, path, 0)
		close(ch)
		hop.Close()
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
				if file.isText {
					slog.Warn("File collision", "hypha", file.hypha,
						"usingFile", h.TextFilePath(), "insteadOf", file.path)
					continue
				}
			case *MediaHypha:
				if !file.isText {
					slog.Warn("File collision", "hypha", file.hypha,
						"usingFile", h.MediaFilePath(), "insteadOf", file.path)
					continue
				}
			}
		}
		var updatedHypha ExistingHypha
		if file.isText {
			updatedHypha = storedHypha.WithTextPath(file.path)
			indexBacklinks(updatedHypha, file.text, newBacklinks)
		} else {
			updatedHypha = storedHypha.WithMediaPath(file.path)
		}
		newByNames[file.hypha] = updatedHypha
	}

	if err != nil {
		slog.Error("Failed to index hyphae", "err", err, "path", path)
		return err
	}

	newHyphae := make([]ExistingHypha, newCount)
	i := 0
	for _, h := range newByNames {
		newHyphae[i] = h
		i++
	}
	slices.SortFunc(newHyphae, Compare)

	indexMutex.Lock()
	hyphae = newHyphae
	byNames = newByNames
	backlinksByName = newBacklinks
	setCount(newCount)
	indexMutex.Unlock()

	slog.Info("Indexed hyphae", "n", newCount)
	return nil
}

func indexBacklinks(
	h ExistingHypha,
	text []byte,
	backlinks map[string]linkSet,
) {
	foundLinks := ExtractHyphaLinksFromBytes(h.CanonicalName(), text)
	for _, link := range foundLinks {
		if _, exists := backlinks[link]; !exists {
			backlinks[link] = make(linkSet)
		}
		backlinks[link][h.CanonicalName()] = struct{}{}
	}
}

// indexHelper finds all hypha files in the full `path` and sends them to the
// channel. Handling of duplicate entries and media and counting them is
// up to the caller.
func indexHelper(
	hop *history.ReadOp,
	ctx context.Context,
	ch chan foundFile,
	path string,
	nestLevel uint,
) error {
	nodes, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf(
			"failed to index directory '%s': %s",
			path, err.Error(),
		)
	}

	for _, node := range nodes {
		// If this hypha looks like it can be a hypha path, go deeper. Do not
		// touch the .git folders for it has an administrative importance!
		if node.IsDir() && IsValidName(node.Name()) && node.Name() != ".git" {
			err = indexHelper(
				hop, ctx, ch,
				filepath.Join(path, node.Name()), nestLevel + 1,
			)
			if err != nil {
				return err
			}
			continue
		}

		var (
			hyphaPartPath           = filepath.Join(path, node.Name())
			hyphaName, isText, skip = mimetype.DataFromFilename(hyphaPartPath)
		)
		if !skip {
			text := []byte(nil)
			if isText {
				text, err = hop.ReadFile(hyphaPartPath)
				if err != nil {
					return fmt.Errorf(
						"failed to index hypha '%s' (%s): %s",
						hyphaName, hyphaPartPath, err.Error(),
					)
				}
			}
			file := foundFile {
				hypha:  hyphaName,
				path:   hyphaPartPath,
				text:   text,
				isText: isText,
			}
			select {
			case <-ctx.Done():
				return nil
			case ch <- file:
			}
		}
	}
	return nil
}
