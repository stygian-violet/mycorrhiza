package migration

import (
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/util"
)

// MigrateSpacesAndNewlinesMaybe checks if the space and newline link migration marker exists. If it exists, nothing is done. If it does not, the migration takes place.
//
// This function writes logs and might terminate the program. Tons of side-effects, stay safe.
func migrateSpacesAndNewlinesMaybe() error {
	markerPath := files.FileInRoot(".mycomarkup-space-and-newline-migration-marker.txt")
	should, err := shouldMigrate(markerPath)
	switch {
	case err != nil:
		return err
	case !should:
		return nil
	}
	err = genericFileMigrator(
		[]string{files.FileInRepo(".gitattributes")},
		"Enable newline conversion in .gitattributes",
		gitAttributeNewlineMigrator,
		"Something went wrong when commiting git attribute migration: ",
	)
	if err != nil {
		return err
	}
	err = genericLineMigrator(
		"Trim spaces and convert newlines to Unix style",
		util.NormalizeText,
		"Something went wrong when commiting space and newline migration: ",
	)
	if err != nil {
		return err
	}
	return createMarker(markerPath, `This file is used to mark that the space and newline migration was made successfully. If this file is deleted, the migration might happen again depending on the version. You should probably not touch this file at all and let it be.`)
}

func gitAttributeNewlineMigrator(text string) string {
	line := "*.myco text\n"
	if strings.Contains(text, line) {
		return text
	}
	return line + text
}
