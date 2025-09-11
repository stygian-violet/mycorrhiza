package history

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"sync/atomic"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
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

func gitgrep(query string, parse func([]byte) (bool, error)) error {
	var (
		limit string
		path string
		ctx context.Context
		cancel context.CancelFunc
	)
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
	if cfg.GrepTimeout > 0 {
		ctx, cancel = context.WithTimeout(process.Context(), cfg.GrepTimeout)
	} else {
		ctx, cancel = context.WithCancel(process.Context())
	}
	defer cancel()
	return gitPipeContext(args, ctx, cancel, parse)
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
	var e *exec.ExitError
	if errors.As(err, &e) {
		return e.ExitCode() != 1
	}
	return true
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

	res := search.NewSearchResults()
	err := gitgrep(query, func(line []byte) (bool, error) {
		err := grepParse(line, res)
		if err != nil {
			return false, err
		}
		parseNext := res.Limit(limit)
		if !parseNext {
			res.Complete = false
		}
		return parseNext, nil
	})

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		slog.Info("Grep timeout", "query", query)
		res.Complete = false
		err = nil
	case grepExitError(err):
		slog.Error("Grep exited with error", "query", query, "err", err)
		res.Complete = false
	default:
		err = nil
	}
	return res, err
}
