// Package files is used to get paths to different files Mycorrhiza uses. Also see cfg.
package files

import (
	"io"
	"os"
	"path/filepath"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/web/static"
)

var paths struct {
	wikiDir             string
	gitRepo             string
	cacheDir            string
	staticFiles         string
	configPath          string
	tokensJSON          string
	userCredentialsJSON string
	categoriesJSON      string
	interwikiJSON       string
}

// HyphaeDir returns the path to hyphae storage.
// A separate function is needed to easily know where a general storage path is
// needed rather than a concrete Git or the whole wiki storage path, so that we
// could easily refactor things later if we'll ever support different storages.
func HyphaeDir() string { return paths.gitRepo }

// GitRepo returns the path to the Git repository of the wiki.
func GitRepo() string { return paths.gitRepo }

// StaticFiles returns the path to static files directory
func StaticFiles() string { return paths.staticFiles }

// ConfigPath returns the path to the config file.
func ConfigPath() string { return paths.configPath }

// TokensJSON returns the path to the JSON user tokens storage.
func TokensJSON() string { return paths.tokensJSON }

// UserCredentialsJSON returns the path to the JSON user credentials storage.
func UserCredentialsJSON() string { return paths.userCredentialsJSON }

// CategoriesJSON returns the path to the JSON categories storage.
func CategoriesJSON() string { return paths.categoriesJSON }

// FileInRoot returns full path for the given filename if it was placed in the root of the wiki structure.
func FileInRoot(filename string) string { return filepath.Join(paths.wikiDir, filename) }

func FileInRepo(filename string) string { return filepath.Join(paths.gitRepo, filename) }

func InterwikiJSON() string { return paths.interwikiJSON }

// PrepareWikiRoot ensures all needed directories and files exist and have
// correct permissions.
func PrepareWikiRoot() error {
	isFirstInit := false
	paths.wikiDir = filepath.FromSlash(cfg.WikiDir)
	if _, err := os.Stat(paths.wikiDir); err != nil && os.IsNotExist(err) {
		isFirstInit = true
	}
	if err := os.MkdirAll(paths.wikiDir, os.ModeDir|0770); err != nil {
		return err
	}

	paths.cacheDir = filepath.Join(paths.wikiDir, "cache")
	if err := os.MkdirAll(paths.cacheDir, os.ModeDir|0770); err != nil {
		return err
	}

	paths.gitRepo = filepath.Join(paths.wikiDir, "wiki.git")
	if err := os.MkdirAll(paths.gitRepo, os.ModeDir|0770); err != nil {
		return err
	}

	paths.staticFiles = filepath.Join(paths.wikiDir, "static")
	if err := os.MkdirAll(paths.staticFiles, os.ModeDir|0770); err != nil {
		return err
	}

	paths.configPath = filepath.Join(paths.wikiDir, "config.ini")
	paths.userCredentialsJSON = filepath.Join(paths.wikiDir, "users.json")

	paths.tokensJSON = filepath.Join(paths.cacheDir, "tokens.json")
	paths.categoriesJSON = filepath.Join(paths.wikiDir, "categories.json")
	paths.interwikiJSON = FileInRoot("interwiki.json")

	// Are we initializing the wiki for the first time?
	if isFirstInit {
		err := firstTimeInit()
		if err != nil {
			return err
		}
	}

	return nil
}

// firstTimeInit takes care of any tasks that only need to happen the first time the wiki is initialized
func firstTimeInit() error {
	static.InitFS(StaticFiles())

	defaultFavicon, err := static.FS.Open("icon/mushroom.png")
	if err != nil {
		return err
	}

	defer defaultFavicon.Close()

	outputFileName := filepath.Join(paths.wikiDir, "static", "favicon.ico")

	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return err
	}

	defer outputFile.Close()

	_, err = io.Copy(outputFile, defaultFavicon)
	if err != nil {
		return err
	}

	return nil
}
