package migration

import (
	"github.com/bouncepaw/mycorrhiza/internal/files"

	"git.sr.ht/~bouncepaw/mycomarkup/v5/tools"
)

// MigrateRocketsMaybe checks if the rocket link migration marker exists. If it exists, nothing is done. If it does not, the migration takes place.
//
// This function writes logs and might terminate the program. Tons of side-effects, stay safe.
func migrateRocketsMaybe() error {
	markerPath := files.FileInRoot(".mycomarkup-rocket-link-migration-marker.txt")
	should, err := shouldMigrate(markerPath)
	switch {
	case err != nil:
		return err
	case !should:
		return nil
	}
	err = genericLineMigrator(
		"Migrate rocket links to the new syntax",
		tools.MigrateRocketLinks,
		"Something went wrong when commiting rocket link migration: ",
	)
	if err != nil {
		return err
	}
	return createMarker(markerPath, `This file is used to mark that the rocket link migration was made successfully. If this file is deleted, the migration might happen again depending on the version. You should probably not touch this file at all and let it be.`)
}
