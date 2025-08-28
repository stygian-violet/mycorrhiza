package migration

import (
	"log/slog"
)

func Migrate() error {
	if err := migrateRocketsMaybe(); err != nil {
		slog.Error("Failed to migrate rocket links", "err", err)
		return err
	}
	if err := migrateHeadingsMaybe(); err != nil {
		slog.Error("Failed to migrate headings", "err", err)
		return err
	}
	if err := migrateSpacesAndNewlinesMaybe(); err != nil {
		slog.Error("Failed to migrate spaces and newlines", "err", err)
		return err
	}
	return nil
}
