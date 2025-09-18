package shroom

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/categories"
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/util"
)

type hyphaToRename struct {
	old  hyphae.ExistingHypha
	new  hyphae.ExistingHypha
}

const redirectionTemplate = `=> %[1]s | 👁️➡️ %[2]s
<= %[1]s | full
`

// Rename renames the old hypha to the new name and makes a history record about that. Call if and only if the user has the permission to rename.
func Rename(
	oldHypha hyphae.ExistingHypha,
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

	hyphaeToRename, err := findHyphaeToRename(
		oldHypha, newName, recursive,
		hop, iop,
	)
	if err != nil {
		hop.Abort()
		iop.Abort()
		return err
	}

	var msg string
	if len(hyphaeToRename) > 1 {
		msg = "Rename ‘%s’ to ‘%s’ recursively"
	} else {
		msg = "Rename ‘%s’ to ‘%s’"
	}
	hop.WithMsg(fmt.Sprintf(msg, oldHypha.CanonicalName(), newName))
	hop.WithFilesRenamed(renamingPairs(hyphaeToRename))
	if hop.HasError() {
		iop.Abort()
		return hop.Abort().Err()
	}

	if leaveRedirections {
		files := make([]string, len(hyphaeToRename))
		for i, h := range hyphaeToRename {
			nh, err := h.leaveRedirection(hop, iop)
			if err != nil {
				hop.Abort()
				iop.Abort()
				return err
			}
			files[i] = nh.TextFilePath()
		}
		hop.WithFiles(files...)
	}

	hop.Apply()
	if hop.HasError() {
		iop.Abort()
		return hop.Err()
	}

	for _, h := range hyphaeToRename {
		categories.RenameHyphaInAllCategories(
			h.old.CanonicalName(),
			h.new.CanonicalName(),
		)
		if leaveRedirections {
			categories.AddHyphaToCategory(
				h.old.CanonicalName(),
				cfg.RedirectionCategory,
			)
		}
	}

	iop.Apply()
	return nil
}

func (hr hyphaToRename) leaveRedirection(
	hop *history.Op,
	iop *hyphae.Op,
) (*hyphae.TextualHypha, error) {
	text := fmt.Sprintf(
		redirectionTemplate,
		hr.new.CanonicalName(),
		util.BeautifulName(hr.new.CanonicalName()),
	)
	h := hyphae.NewTextualHypha(hr.old.CanonicalName())
	err := hop.WriteFile(h.TextFilePath(), []byte(text))
	if err != nil {
		return nil, err
	}
	iop.WithHyphaCreated(h, text)
	return h, nil
}

func findHyphaeToRename(
	superhypha hyphae.ExistingHypha,
	newName string,
	recursive bool,
	hop *history.Op,
	iop *hyphae.Op,
) ([]hyphaToRename, error) {
	if iop.Exists(newName) {
		return nil, fmt.Errorf("name '%s' is already taken", newName)
	}
	newSuperhypha := superhypha.WithName(newName)
	hyphaList := []hyphaToRename{{
		old: superhypha,
		new: newSuperhypha,
	}}
	text, err := superhypha.Text(hop)
	if err != nil {
		return nil, err
	}
	iop.WithHyphaRenamed(superhypha, newSuperhypha, text)
	if recursive {
		oldName := superhypha.CanonicalName()
		for h := range iop.YieldSubhyphae(superhypha) {
			name := strings.Replace(h.CanonicalName(), oldName, newName, 1)
			if iop.Exists(newName) {
				return nil, fmt.Errorf("name '%s' is already taken", name)
			}
			text, err := h.Text(hop)
			if err != nil {
				return nil, err
			}
			nh := h.WithName(name)
			iop.WithHyphaRenamed(h, nh, text)
			hyphaList = append(hyphaList, hyphaToRename{
				old: h,
				new: nh,
			})
		}
	}
	return hyphaList, nil
}

func renamingPairs(hyphaeToRename []hyphaToRename) map[string]string {
	renameMap := make(map[string]string)
	for _, h := range hyphaeToRename {
		oldFiles := h.old.FilePaths()
		newFiles := h.new.FilePaths()
		n := len(oldFiles)
		for i := 0; i < n; i++ {
			renameMap[oldFiles[i]] = newFiles[i]
		}
	}
	return renameMap
}
