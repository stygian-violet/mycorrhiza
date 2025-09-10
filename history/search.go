package history

import (
	"bufio"
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"sync/atomic"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/internal/mimetype"
	"github.com/bouncepaw/mycorrhiza/internal/process"
	"github.com/bouncepaw/mycorrhiza/internal/search"
)

var (
	ErrGrepLimit error = errors.New("grep process limit exceeded")
	ErrGrepParse error = errors.New("failed to parse grep output")

	grepCount atomic.Int32
	color = regexp.MustCompile(`\033\[[0-9]*(;[0-9]+)?m`)
)

func gitgrep(query string, ctx context.Context) *exec.Cmd {
	var limit string
	var path string
	if cfg.GrepMatchLimitPerHypha > 0 {
		limit = strconv.FormatUint(uint64(cfg.GrepMatchLimitPerHypha), 10)
	} else {
		limit = "-1"
	}
	if cfg.GrepIgnoreMedia {
		path = "*.myco"
	} else {
		path = "*"
	}
	args := []string{
		"grep", "-i", "-I", "-F", "--color",
		"-m", limit,
		"-e", query,
		"--", ":!.*", path,
	}
	slog.Info(gitstr(args...))
	cmd := exec.CommandContext(ctx, gitpath, args...)
	cmd.Dir = files.HyphaeDir()
	cmd.Env = append(cmd.Environ(), gitEnv...)
	return cmd
}

func grepCountInc() bool {
	for {
		count := grepCount.Load()
		if uint(count) >= cfg.GrepProcessLimit {
			return false
		}
		if grepCount.CompareAndSwap(count, count + 1) {
			return true
		}
	}
}

func grepCountDec() {
	grepCount.Add(-1)
}

func grepExitError(err error) bool {
	if err == nil {
		return false
	}
	switch e := err.(type) {
	case *exec.ExitError:
		return e.ExitCode() != 1
	default:
		return true
	}
}

func grepParse(line []byte, res *search.SearchResults) error {
	if len(line) == 0 {
		return nil
	}
	parts := color.Split(string(line), -1)
	if len(parts) < 5 || parts[0] != "" || parts[2] != "" || parts[3] != ":" {
		slog.Error("Failed to parse grep output", "line", line, "parts", parts)
		return ErrGrepParse
	}
	fname := parts[1]
	parts = parts[4:]
	hyphaName, _, skip := mimetype.DataFromFilename(fname)
	if !skip {
		res.Append(hyphaName, parts, cfg.FullTextLineLength, cfg.GrepMatchLimitPerHypha)
	}
	return nil
}

func Grep(query string, limit int) (*search.SearchResults, error) {
	if limit == 0 {
		return search.NewSearchResults(), nil
	}
	if cfg.GrepProcessLimit > 0 {
		if ! grepCountInc() {
			return nil, ErrGrepLimit
		}
		defer grepCountDec()
	}
	gitMutex.RLock()
	defer gitMutex.RUnlock()
	var (
		ctx context.Context
		cancel context.CancelFunc
	)
	if cfg.GrepTimeout > 0 {
		ctx, cancel = context.WithTimeout(process.Context(), cfg.GrepTimeout)
	} else {
		ctx, cancel = context.WithCancel(process.Context())
	}
	defer cancel()
	cmd := gitgrep(query, ctx)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("Failed to pipe grep stdout", "err", err)
		return nil, err
	}
	err = cmd.Start()
	if err != nil {
		slog.Error("Failed to start grep", "err", err)
		stdout.Close()
		return nil, err
	}
	res := search.NewSearchResults()
	var res2 error = nil
	limited := false
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		err = grepParse(line, res)
		if err != nil {
			cancel()
			res2 = err
			break
		}
		if !res.Limit(limit) {
			limited = true
			cancel()
			break
		}
	}
	err = scanner.Err()
	if err != nil {
		slog.Error("Grep scanner error", "err", err)
		res.Complete = false
		if res2 == nil {
			res2 = err
		}
	}
	err = cmd.Wait()
	switch {
	case limited:
	case ctx.Err() == context.DeadlineExceeded:
		slog.Info("Grep timeout")
		res.Complete = false
	case grepExitError(err):
		slog.Error("Grep exited with error", "err", err)
		res.Complete = false
		if res2 == nil {
			res2 = err
		}
	}
	return res, res2
}
