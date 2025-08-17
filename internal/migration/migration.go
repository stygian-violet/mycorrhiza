// Package migration holds the utilities for migrating from older incompatible Mycomarkup versions.
//
// Migrations are meant to be removed couple of versions after being introduced.
//
// Available migrations:
//   - Rocket links
//   - Headings
package migration

import (
	"io"
	"io/ioutil"
	"log/slog"
	"os"
	"strings"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/user"
)

func shouldMigrate(markerPath string) bool {
	file, err := os.Open(markerPath)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		slog.Error("Failed to check if migration is needed", "markerPath", markerPath, "err", err)
		os.Exit(1)
	}
	_ = file.Close()
	return false
}

func createMarker(markerPath string, contents string) {
	err := ioutil.WriteFile(markerPath, []byte(contents), 0660)
	if err != nil {
		slog.Error("Failed to create migration marker", "markerPath", markerPath, "err", err)
		os.Exit(1)
	}
}

func genericLineMigrator(
	commitMessage string,
	migrator func(string) string,
	commitErrorMessage string,
) {
	var (
		hop = history.
			Operation(history.TypeMarkupMigration).
			WithMsg(commitMessage).
			WithUser(user.WikimindUser())
		mycoFiles = []string{}
	)

	slog.Info(commitMessage)

	for hypha := range hyphae.FilterHyphaeWithText(hyphae.YieldExistingHyphae()) {
		/// Open file, read from file, modify file. If anything goes wrong, scream and shout.

		file, err := os.OpenFile(hypha.TextFilePath(), os.O_RDWR, 0660)
		if err != nil {
			hop.WithErrAbort(err)
			slog.Error("Failed to open text part file", "path", hypha.TextFilePath(), "err", err)
			os.Exit(1)
		}

		var buf strings.Builder
		_, err = io.Copy(&buf, file)
		if err != nil {
			hop.WithErrAbort(err)
			_ = file.Close()
			slog.Error("Failed to read text part file", "path", hypha.TextFilePath(), "err", err)
			os.Exit(1)
		}

		var (
			oldText = buf.String()
			newText = migrator(oldText)
		)
		if oldText != newText { // This file right here is being migrated for real.
			mycoFiles = append(mycoFiles, hypha.TextFilePath())
			hop.SetFilesChanged()

			err = file.Truncate(0)
			if err != nil {
				hop.WithErrAbort(err)
				_ = file.Close()
				slog.Error("Failed to truncate text part file", "path", hypha.TextFilePath(), "err", err)
				os.Exit(1)
			}

			_, err = file.Seek(0, 0)
			if err != nil {
				hop.WithErrAbort(err)
				_ = file.Close()
				slog.Error("Failed to seek in text part file", "path", hypha.TextFilePath(), "err", err)
				os.Exit(1)
			}

			_, err = file.WriteString(newText)
			if err != nil {
				hop.WithErrAbort(err)
				_ = file.Close()
				slog.Error("Failed to write to text part file", "path", hypha.TextFilePath(), "err", err)
				os.Exit(1)
			}
		}
		_ = file.Close()
	}

	if len(mycoFiles) == 0 {
		hop.Abort()
		return
	}

	if hop.WithFiles(mycoFiles...).Apply().HasError() {
		slog.Error(commitErrorMessage + hop.ErrorText())
		os.Exit(1)
	}

	slog.Info("Migrated Mycomarkup documents", "n", len(mycoFiles))
}

func genericFileMigrator(
	paths []string,
	commitMessage string,
	migrator func(string) string,
	commitErrorMessage string,
) {
	var (
		hop = history.
			Operation(history.TypeMarkupMigration).
			WithMsg(commitMessage).
			WithUser(user.WikimindUser())
		changedFiles = []string{}
	)

	slog.Info(commitMessage)

	for _, path := range paths {
		/// Open file, read from file, modify file. If anything goes wrong, scream and shout.
		var (
			exists bool = true
			oldText string
		)

		if _, err := os.Stat(path); os.IsNotExist(err) {
			exists = false
			hop.SetFilesChanged()
		}

		file, err := os.OpenFile(path, os.O_CREATE | os.O_RDWR, 0660)
		if err != nil {
			hop.WithErrAbort(err)
			slog.Error("Failed to open file", "path", path, "err", err)
			os.Exit(1)
		}

		if exists {
			var buf strings.Builder
			_, err = io.Copy(&buf, file)
			if err != nil {
				hop.WithErrAbort(err)
				_ = file.Close()
				slog.Error("Failed to read file", "path", path, "err", err)
				os.Exit(1)
			}
			oldText = buf.String()
		} else {
			oldText = ""
		}

		newText := migrator(oldText)

		if !exists || oldText != newText { // This file right here is being migrated for real.
			changedFiles = append(changedFiles, path)
			hop.SetFilesChanged()

			err = file.Truncate(0)
			if err != nil {
				hop.WithErrAbort(err)
				_ = file.Close()
				slog.Error("Failed to truncate file", "path", path, "err", err)
				os.Exit(1)
			}

			_, err = file.Seek(0, 0)
			if err != nil {
				hop.WithErrAbort(err)
				_ = file.Close()
				slog.Error("Failed to seek in file", "path", path, "err", err)
				os.Exit(1)
			}

			_, err = file.WriteString(newText)
			if err != nil {
				hop.WithErrAbort(err)
				_ = file.Close()
				slog.Error("Failed to write to text part file", "path", path, "err", err)
				os.Exit(1)
			}
		}
		_ = file.Close()
	}

	if len(changedFiles) == 0 {
		hop.Abort()
		return
	}

	if hop.WithFiles(changedFiles...).Apply().HasError() {
		slog.Error(commitErrorMessage + hop.ErrorText())
		os.Exit(1)
	}

	slog.Info("Migrated files", "n", len(changedFiles))
}
