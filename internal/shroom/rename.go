package shroom

import (
	"errors"
	"fmt"
	"iter"
	"strings"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/categories"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/util"
)

const redirectionTemplate = `=> %[1]s | 👁️➡️ %[2]s
<= %[1]s | full
`

var ErrRenameEmpty = errors.New("nothing to rename")

// Rename renames the old hypha to the new name and makes a history record about that. Call if and only if the user has the permission to rename.
func Rename(
	oldHypha hyphae.Hypha,
	newName string,
	recursive bool,
	leaveRedirections bool,
	u *user.User,
) error {
	oldName := oldHypha.CanonicalName()

	switch {
	case newName == "":
		return errors.New("ui.rename_noname_tip")
	case !hyphae.IsValidName(newName):
		return errors.New("ui.rename_badname_tip") // FIXME: There is a bug related to this.
	case oldName == newName:
		return nil
	}

	hop := history.Operation().WithUser(u)
	iop := hyphae.IndexOperation()

	hyphaeToRename := yieldHyphaeToRename(oldHypha, newName, recursive, iop)
	names, files, err := renamingPairs(hyphaeToRename, hop, iop)
	if len(names) == 0 && err == nil {
		err = ErrRenameEmpty
	}
	if err != nil {
		hop.Abort()
		iop.Abort()
		return err
	}

	var msg string
	if len(names) > 1 || names[0].From() != oldName {
		msg = "Rename ‘%s’ to ‘%s’ recursively"
	} else {
		msg = "Rename ‘%s’ to ‘%s’"
	}
	hop.WithMsg(fmt.Sprintf(msg, oldHypha.CanonicalName(), newName)).
		WithFilesRenamed(files...)
	if hop.HasError() {
		hop.Abort()
		iop.Abort()
		return hop.Abort().Err()
	}

	if leaveRedirections {
		redirections := make([]string, len(names))
		for i, pair := range names {
			h, err := leaveRedirection(pair, hop, iop)
			if err != nil {
				hop.Abort()
				iop.Abort()
				return err
			}
			redirections[i] = h.TextFilePath()
		}
		hop.WithFiles(redirections...)
	}

	hop.Apply()
	if hop.HasError() {
		iop.Abort()
		return hop.Err()
	}

	categories.RenameHyphaeInAllCategories(leaveRedirections, names...)
	iop.Apply()
	return nil
}

func leaveRedirection(
	pair util.RenamingPair[string],
	hop *history.Op,
	iop *hyphae.Op,
) (*hyphae.TextualHypha, error) {
	text := fmt.Sprintf(
		redirectionTemplate,
		pair.To(),
		util.BeautifulName(pair.To()),
	)
	h := hyphae.NewTextualHypha(pair.From())
	err := hop.WriteFile(h.TextFilePath(), []byte(text))
	if err != nil {
		return nil, err
	}
	iop.WithHyphaCreated(h, text)
	return h, nil
}

func yieldHyphaeToRename(
	superhypha hyphae.Hypha,
	newName string,
	recursive bool,
	iop *hyphae.Op,
) iter.Seq[hyphae.RenamingPair] {
	return func(yield func(hyphae.RenamingPair) bool) {
		if sh, ok := superhypha.(hyphae.ExistingHypha); ok {
			newSuperhypha := sh.WithName(newName)
			rp := util.NewRenamingPair(sh, newSuperhypha)
			if !yield(rp) {
				return
			}
		}
		if !recursive {
			return
		}
		oldName := superhypha.CanonicalName()
		for h := range iop.YieldSubhyphae(superhypha) {
			name := strings.Replace(h.CanonicalName(), oldName, newName, 1)
			rp := util.NewRenamingPair(h, h.WithName(name))
			if !yield(rp) {
				return
			}
		}
	}
}

func renamingPairs(
	pairs iter.Seq[hyphae.RenamingPair],
	hop *history.Op,
	iop *hyphae.Op,
) (names []util.RenamingPair[string], files []util.RenamingPair[string], err error) {
	for pair := range pairs {
		oldName := pair.From().CanonicalName()
		newName := pair.To().CanonicalName()
		if iop.Exists(newName) {
			return nil, nil, fmt.Errorf("name '%s' is already taken", newName)
		}
		text, err := pair.From().Text(hop)
		if err != nil {
			return nil, nil, err
		}

		iop.WithHyphaRenamedPair(pair, text)
		names = append(names, util.NewRenamingPair(oldName, newName))

		oldFiles := pair.From().FilePaths()
		newFiles := pair.To().FilePaths()
		n := len(oldFiles)
		if n != len(newFiles) {
			return nil, nil, fmt.Errorf(
				"renaming pair has different number of files: %v -> %v",
				oldFiles, newFiles,
			)
		}
		for i := 0; i < n; i++ {
			files = append(files, util.NewRenamingPair(oldFiles[i], newFiles[i]))
		}
	}
	return
}
