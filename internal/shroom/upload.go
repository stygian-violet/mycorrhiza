package shroom

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/backlinks"
	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/mimetype"
	"github.com/bouncepaw/mycorrhiza/internal/user"
)

func historyMessageForTextUpload(h hyphae.Hypha, userMessage string) string {
	var verb string
	switch h.(type) {
	case *hyphae.EmptyHypha:
		verb = "Create"
	default:
		verb = "Edit"
	}

	if userMessage == "" {
		return fmt.Sprintf("%s ‘%s’", verb, h.CanonicalName())
	}
	return fmt.Sprintf("%s ‘%s’: %s", verb, h.CanonicalName(), userMessage)
}

func writeTextToDisk(h hyphae.ExistingHypha, data string, hop *history.Op) error {
	hop.SetFilesChanged()
	if err := hyphae.WriteToMycoFile(h, []byte(data)); err != nil {
		return err
	}
	return hop.WithFiles(h.TextFilePath()).Error
}

// UploadText edits the hypha's text part and makes a history record about that.
func UploadText(h hyphae.Hypha, text string, userMessage string, u *user.User) error {
	hop := history.
		Operation(history.TypeEditText).
		WithMsg(historyMessageForTextUpload(h, userMessage)).
		WithUser(u)

	// Privilege check
	if !u.CanProceed("upload-text") {
		hop.Abort()
		rejectEditLog(h, u, "no rights")
		return errors.New("ui.act_no_rights")
	}

	// Hypha name exploit check
	if !hyphae.IsValidName(h.CanonicalName()) {
		hop.Abort()
		// We check for the name only. I suppose the filepath would be valid as well.
		return errors.New("invalid hypha name")
	}

	// text := util.NormalizeText(text)

	oldText, err := hyphae.FetchMycomarkupFile(h)
	if err != nil {
		hop.Abort()
		return err
	}

	if text == oldText {
		// No changes! Just like cancel button
		hop.Abort()
		return nil
	}

	// At this point, we have a savable user-generated Mycomarkup document. Gotta save it.

	insert := false
	var H hyphae.ExistingHypha = nil

	switch h := h.(type) {
	case *hyphae.EmptyHypha:
		parts := []string{files.HyphaeDir()}
		parts = append(parts, strings.Split(h.CanonicalName()+".myco", "\\")...)
		H = hyphae.ExtendEmptyToTextual(h, filepath.Join(parts...))
		insert = true
	case *hyphae.MediaHypha:
		H = h
	case *hyphae.TextualHypha:
		H = h
	}

	err = writeTextToDisk(H, text, hop)
	if err != nil {
		hop.Abort()
		return err
	}

	backlinks.UpdateBacklinksAfterEdit(h, oldText)
	if insert {
		hyphae.Insert(H)
	}

	hop.Apply()
	if hop.HasError() {
		Reindex()
	}
	return hop.Error
}

func historyMessageForMediaUpload(h hyphae.Hypha, mime string) string {
	return fmt.Sprintf("Upload media for ‘%s’ with type ‘%s’", h.CanonicalName(), mime)
}

// writeMediaToDisk saves the given data with the given mime type for the given hypha to the disk and returns the path to the saved file and an error, if any.
func writeMediaToDisk(h hyphae.Hypha, mime string, data []byte) (string, error) {
	var (
		ext = mimetype.ToExtension(mime)
		// That's where the file will go

		uploadedFilePath = filepath.Join(append([]string{files.HyphaeDir()}, strings.Split(h.CanonicalName()+ext, "\\")...)...)
	)

	if err := os.MkdirAll(filepath.Dir(uploadedFilePath), os.ModeDir|0770); err != nil {
		return uploadedFilePath, err
	}

	if err := os.WriteFile(uploadedFilePath, data, 0660); err != nil {
		return uploadedFilePath, err
	}
	return uploadedFilePath, nil
}

// UploadBinary edits the hypha's media part and makes a history record about that.
func UploadBinary(h hyphae.Hypha, mime string, file multipart.File, u *user.User) error {
	hop := history.
		Operation(history.TypeEditBinary).
		WithMsg(historyMessageForMediaUpload(h, mime)).
		WithUser(u)

	// Privilege check
	if !u.CanProceed("upload-binary") {
		hop.Abort()
		rejectUploadMediaLog(h, u, "no rights")
		return errors.New("ui.act_no_rights")
	}

	// Hypha name exploit check
	if !hyphae.IsValidName(h.CanonicalName()) {
		hop.Abort()
		// We check for the name only. I suppose the filepath would be valid as well.
		return errors.New("invalid hypha name")
	}

	data, err := io.ReadAll(file)
	if err != nil {
		hop.Abort()
		return err
	}

	// Empty data check
	if len(data) == 0 {
		hop.Abort()
		return errors.New("No data passed")
	}

	// At this point, we have a savable media document. Gotta save it.
	hop.SetFilesChanged()
	uploadedFilePath, err := writeMediaToDisk(h, mime, data)
	if err != nil {
		hop.Abort()
		return err
	}

	var H *hyphae.MediaHypha = nil
	insert := false

	switch h := h.(type) {
	case *hyphae.EmptyHypha:
		insert = true
		H = hyphae.ExtendEmptyToMedia(h, uploadedFilePath)
	case *hyphae.TextualHypha:
		insert = true
		H = hyphae.ExtendTextualToMedia(h, uploadedFilePath)
	case *hyphae.MediaHypha: // If this is not the first media the hypha gets
		H = h
		prevFilePath := h.MediaFilePath()
		if prevFilePath != uploadedFilePath {
			hop.WithFilesRemoved(prevFilePath)
			slog.Info("Move file", "from", prevFilePath, "to", uploadedFilePath)
		}
	}

	hop.WithFiles(uploadedFilePath)
	if hop.HasError() {
		hop.Abort()
		return hop.Error
	}

	if insert {
		hyphae.Insert(H)
	} else {
		H.SetMediaFilePath(uploadedFilePath)
	}

	hop.Apply()
	if hop.HasError() {
		Reindex()
	}
	return hop.Error
}
