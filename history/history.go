// Package history provides a git wrapper.
package history

import (
	"bytes"
	"log/slog"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/util"
)

// Path to git executable. Set at init()
var gitpath string

var renameMsgPattern = regexp.MustCompile(`^Rename ‘(.*)’ to ‘.*’`)

var gitEnv = []string{"GIT_COMMITTER_NAME=wikimind", "GIT_COMMITTER_EMAIL=wikimind@mycorrhiza"}

// Start finds git and initializes git credentials.
func Start() error {
	path, err := exec.LookPath("git")
	if err != nil {
		slog.Error("Could not find the Git executable. Check your $PATH.")
		return err
	}
	gitpath = path
	return nil
}

// InitGitRepo checks a Git repository and initializes it if necessary.
func InitGitRepo() error {
	// Detect if the Git repo directory is a Git repository
	isGitRepo := true
	buf, err := gitsh("rev-parse", "--git-dir")
	if err != nil {
		isGitRepo = false
	}
	if isGitRepo {
		gitDir := buf.String()
		if filepath.IsAbs(gitDir) && !filepath.HasPrefix(gitDir, files.HyphaeDir()) {
			isGitRepo = false
		}
	}
	if !isGitRepo {
		slog.Info("Initializing Git repo", "path", files.HyphaeDir())
		if _, err := gitsh("init"); err != nil {
			return err
		}
		if _, err := gitsh("config", "core.quotePath", "false"); err != nil {
			return err
		}
	}
	return nil
}

func gitstr(args ...string) string {
	return strings.Join(append([]string{"> git"}, args...), " ");
}

// I pronounce it as [gɪt͡ʃ].
// gitsh is async-safe, therefore all other git-related functions in this module are too.
func gitsh(args ...string) (out bytes.Buffer, err error) {
	slog.Info(gitstr(args...))
	cmd := exec.Command(gitpath, args...)
	cmd.Dir = files.HyphaeDir()
	cmd.Env = append(cmd.Environ(), gitEnv...)

	b, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("Git command failed", "args", args, "err", err, "output", string(b))
	}
	return *bytes.NewBuffer(b), err
}

func gitReset() error {
	slog.Info("Resetting Git working directory")
	var ret error = nil
	if _, err := gitsh("reset", "--hard"); err != nil {
		ret = err
	}
	if _, err := gitsh("clean", "-d", "-f"); err != nil {
		ret = err
	}
	if ret != nil {
		slog.Error("Failed to reset working tree")
	}
	return ret
}

// Rename renames from `from` to `to` using `git mv`.
func Rename(from, to string) error {
	slog.Info("Renaming file with git mv",
		"from", util.ShorterPath(from),
		"to", util.ShorterPath(to))
	_, err := gitsh("mv", "--force", from, to)
	return err
}
