package shroom

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/mimetype"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/util"
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

// UploadText edits the hypha's text part and makes a history record about that.
func UploadText(h hyphae.Hypha, text string, userMessage string, u *user.User) error {
	// Hypha name exploit check
	if !hyphae.IsValidName(h.CanonicalName()) {
		// We check for the name only. I suppose the filepath would be valid as well.
		return errors.New("invalid hypha name")
	}

	hop := history.
		Operation().
		WithMsg(historyMessageForTextUpload(h, userMessage)).
		WithUser(u)

	oldText, err := h.Text(hop)
	if err != nil {
		hop.Abort()
		return err
	}

	text = util.NormalizeText(text)
	if text == oldText {
		// No changes! Just like cancel button
		hop.Abort()
		return nil
	}

	iop := hyphae.IndexOperation()

	path := h.TextFilePath()
	nh := h.WithTextPath(path)
	if he, exists := h.(hyphae.ExistingHypha); exists {
		iop.WithHyphaTextChanged(he, oldText, nh, text)
	} else {
		iop.WithHyphaCreated(nh, text)
	}

	err = hop.WriteFile(path, []byte(text))
	if err != nil {
		hop.Abort()
		iop.Abort()
		return err
	}

	hop.WithFiles(path).Apply()
	if hop.HasError() {
		iop.Abort()
		return hop.Err()
	}

	iop.Apply()
	return nil
}

func historyMessageForMediaUpload(h hyphae.Hypha, mime string) string {
	return fmt.Sprintf("Upload media for ‘%s’ with type ‘%s’", h.CanonicalName(), mime)
}

// UploadBinary edits the hypha's media part and makes a history record about that.
func UploadBinary(
	h hyphae.Hypha,
	filename string,
	mime string,
	file io.ReadSeeker,
	u *user.User,
) error {
	// Hypha name exploit check
	if !hyphae.IsValidName(h.CanonicalName()) {
		// We check for the name only. I suppose the filepath would be valid as well.
		return errors.New("invalid hypha name")
	}
	size, err := util.FileSize(file)
	switch {
	case err != nil:
		return err
	case size <= 0:
		return errors.New("No data passed")
	}

	hop := history.
		Operation().
		WithMsg(historyMessageForMediaUpload(h, mime)).
		WithUser(u)
	iop := hyphae.IndexOperation()

	// At this point, we have a savable media document. Gotta save it.
	ext := mimetype.ToExtension(filename, mime)
	uploadedFilePath := hyphae.FilePath(h.CanonicalName()) + ext

	if ht, ok := h.(*hyphae.MediaHypha); ok {
		prevFilePath := ht.MediaFilePath()
		if prevFilePath != uploadedFilePath {
			hop.WithFilesRemoved(prevFilePath)
			slog.Info("Move file", "from", prevFilePath, "to", uploadedFilePath)
		}
	}

	nh := h.WithMediaPath(uploadedFilePath)
	if he, ok := h.(hyphae.ExistingHypha); ok {
		iop.WithHyphaMediaChanged(he, nh)
	} else {
		iop.WithHyphaCreated(nh, "")
	}

	err = hop.CopyFile(uploadedFilePath, file)
	if err != nil {
		hop.Abort()
		iop.Abort()
		return err
	}
	hop.WithFiles(uploadedFilePath)

	hop.Apply()
	if hop.HasError() {
		iop.Abort()
		return hop.Err()
	}

	iop.Apply()
	return nil
}
