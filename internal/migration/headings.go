package migration

import (
	"github.com/bouncepaw/mycorrhiza/internal/files"

	"git.sr.ht/~bouncepaw/mycomarkup/v5/tools"
)

func migrateHeadingsMaybe() error {
	markerPath := files.FileInRoot(".mycomarkup-heading-migration-marker.txt")
	should, err := shouldMigrate(markerPath)
	switch {
	case err != nil:
		return err
	case !should:
		return nil
	}
	err = genericLineMigrator(
		"Migrate headings to the new syntax",
		tools.MigrateHeadings,
		"Something went wrong when commiting heading migration: ",
	)
	if err != nil {
		return err
	}
	return createMarker(markerPath, `This file is used to mark that the heading migration was successful. If this file is deleted, the migration might happen again depending on the version. You should probably not touch this file at all and let it be.`)
}
