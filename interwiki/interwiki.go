// Package interwiki provides interwiki capabilities. Most of them, at least.
package interwiki

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sync"

	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/util"

	"git.sr.ht/~bouncepaw/mycomarkup/v5/options"
)

var (
	entries    []*Wiki
	byName     map[string]*Wiki
	indexMutex sync.RWMutex
	fileMutex  sync.Mutex
)

func Init() error {
	newEntries, err := readInterwiki()
	if err != nil {
		slog.Error("Failed to read interwiki", "err", err)
		return err
	}

	newByName := make(map[string]*Wiki)
	for _, wiki := range newEntries {
		if err := canReplace(newByName, EmptyWiki(), wiki); err != nil {
			slog.Error("Failed to add interwiki entry", "wiki", wiki, "err", err)
			return err
		}
		for name := range wiki.Names() {
			newByName[name] = wiki
		}
	}

	slices.SortFunc(newEntries, Compare)

	indexMutex.Lock()
	entries = newEntries
	byName = newByName
	count := len(newEntries)
	indexMutex.Unlock()

	slog.Info("Indexed interwiki map", "n", count)
	return nil
}

func ByName(name string) *Wiki {
	name = util.CanonicalName(name)
	indexMutex.RLock()
	wiki := byName[name]
	indexMutex.RUnlock()
	if wiki == nil {
		wiki = EmptyWiki()
	}
	return wiki
}

func Entries() []*Wiki {
	indexMutex.RLock()
	res := make([]*Wiki, len(entries))
	copy(res, entries)
	indexMutex.RUnlock()
	return res
}

func HrefLinkFormatFor(prefix string) (string, options.InterwikiError) {
	wiki := ByName(prefix)
	if wiki.IsEmpty() {
		return "", options.UnknownPrefix
	}
	return wiki.LinkHrefFormat(), options.Ok
}

func ImgSrcFormatFor(prefix string) (string, options.InterwikiError) {
	wiki := ByName(prefix)
	if wiki.IsEmpty() {
		return "", options.UnknownPrefix
	}
	return wiki.ImgSrcFormat(), options.Ok
}

func ReplaceEntry(oldWiki *Wiki, newWiki *Wiki) error {
	if oldWiki.IsEmpty() && newWiki.IsEmpty() {
		return nil
	}
	indexMutex.Lock()
	defer indexMutex.Unlock()
	if err := canReplace(byName, oldWiki, newWiki); err != nil {
		return err
	}
	newEntries := make([]*Wiki, len(entries))
	copy(newEntries, entries)
	switch {
	case newWiki.IsEmpty():
		newEntries = util.DeleteSorted(newEntries, Compare, oldWiki)
	case oldWiki.IsEmpty():
		newEntries = util.InsertSorted(newEntries, Compare, newWiki)
	default:
		newEntries = util.ReplaceSorted(newEntries, Compare, oldWiki, newWiki)
	}
	if err := saveInterwikiJson(newEntries); err != nil {
		return err
	}
	entries = newEntries
	for name := range oldWiki.Names() {
		delete(byName, name)
	}
	for name := range newWiki.Names() {
		byName[name] = newWiki
	}
	return nil
}

func AddEntry(newWiki *Wiki) error {
	return ReplaceEntry(EmptyWiki(), newWiki)
}

func DeleteEntry(oldWiki *Wiki) error {
	return ReplaceEntry(oldWiki, EmptyWiki())
}

func canReplace(
	byname map[string]*Wiki,
	oldWiki *Wiki,
	newWiki *Wiki,
) error {
	for name := range newWiki.Names() {
		existingWiki, exists := byname[name]
		if exists && existingWiki != oldWiki {
			return fmt.Errorf(
				"wiki name '%s' of %s is already taken by %s",
				name, newWiki, existingWiki,
			)
		}
	}
	return nil
}

func readInterwiki() ([]*Wiki, error) {
	fileMutex.Lock()
	fileContents, err := os.ReadFile(files.InterwikiJSON())
	fileMutex.Unlock()
	switch {
	case os.IsNotExist(err):
		return nil, nil
	case err != nil:
		return nil, err
	}
	record := []*Wiki(nil)
	if err = json.Unmarshal(fileContents, &record); err != nil {
		return nil, err
	}
	return record, nil
}

func saveInterwikiJson(wikis []*Wiki) error {
	data, err := json.MarshalIndent(wikis, "", "\t")
	if err != nil {
		slog.Error("Failed to marshal interwiki entries", "err", err)
		return err
	}

	fileMutex.Lock()
	err = os.WriteFile(files.InterwikiJSON(), data, 0660)
	fileMutex.Unlock()
	if err != nil {
		slog.Error("Failed to write interwiki.json", "err", err)
		return err
	}

	slog.Info("Saved interwiki.json")
	return nil
}
