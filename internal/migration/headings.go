package migration

import (
	"github.com/bouncepaw/mycorrhiza/internal/files"

	"git.sr.ht/~bouncepaw/mycomarkup/v5/tools"
)

func MigrateHeadingsMaybe() {
	markerPath := files.FileInRoot(".mycomarkup-heading-migration-marker.txt")
	if !shouldMigrate(markerPath) {
		return
	}
	genericLineMigrator(
		"Migrate headings to the new syntax",
		tools.MigrateHeadings,
		"Something went wrong when commiting heading migration: ",
	)
	createMarker(markerPath, `This file is used to mark that the heading migration was successful. If this file is deleted, the migration might happen again depending on the version. You should probably not touch this file at all and let it be.`)
}
