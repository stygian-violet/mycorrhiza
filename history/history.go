// Package history provides a git wrapper.
package history

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/internal/process"
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
		gitDir := string(buf)
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
func gitsh(args ...string) ([]byte, error) {
	slog.Info(gitstr(args...))
	cmd := exec.Command(gitpath, args...)
	cmd.Dir = files.HyphaeDir()
	cmd.Env = append(cmd.Environ(), gitEnv...)

	out, err := cmd.CombinedOutput()
	if err != nil {
		outStr := string(out)
		if (len(args) > 0 &&
			args[0] == "commit" &&
			strings.Contains(outStr, "nothing to commit")) {
			slog.Info("Nothing to commit", "output", outStr)
			err = nil
		} else {
			slog.Error(
				"Git command failed",
				"args", args, "err", err, "output", outStr,
			)
		}
	}
	return out, err
}

func gitPipeStart(
	ctx context.Context,
	args... string,
) (*exec.Cmd, io.ReadCloser, error) {
	slog.Info(gitstr(args...))
	cmd := exec.CommandContext(ctx, gitpath, args...)
	cmd.Dir = files.HyphaeDir()
	cmd.Env = append(cmd.Environ(), gitEnv...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Failed to pipe git stdout", "err", err)
		return nil, stdout, err
	}
	cmd.Stderr = cmd.Stdout
	err = cmd.Start()
	if err != nil {
		slog.Error("Failed to start git", "args", args, "err", err)
		stdout.Close()
		return nil, stdout, err
	}
	return cmd, stdout, nil
}

func gitPipeContext(
	args []string,
	ctx context.Context,
	cancel context.CancelFunc,
	parse func([]byte) (bool, error),
) error {
	cmd, stdout, err := gitPipeStart(ctx, args...)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	parseNext, parseErr := true, error(nil)
	for scanner.Scan() {
		line := scanner.Bytes()
		parseNext, parseErr = parse(line)
		if parseErr != nil {
			cancel()
			break
		}
		if !parseNext {
			cancel()
			break
		}
	}
	err = scanner.Err()
	if err != nil {
		slog.Error("Git scanner error", "args", args, "err", err)
		if parseErr == nil {
			parseErr = err
		}
	}
	err = cmd.Wait()
	switch {
	case parseErr != nil:
		return parseErr
	case !parseNext:
		return nil
	case ctx.Err() != nil:
		return ctx.Err()
	default:
		return err
	}
}

func gitPipe(args []string, parse func([]byte) (bool, error)) error {
	ctx, cancel := context.WithCancel(process.Context())
	defer cancel()
	return gitPipeContext(args, ctx, cancel, parse)
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
