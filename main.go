// Command mycorrhiza is a program that runs a mycorrhiza wiki.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bouncepaw/mycorrhiza/history"
	"github.com/bouncepaw/mycorrhiza/internal/categories"
	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/internal/hyphae"
	"github.com/bouncepaw/mycorrhiza/internal/migration"
	"github.com/bouncepaw/mycorrhiza/internal/process"
	"github.com/bouncepaw/mycorrhiza/internal/shroom"
	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/internal/version"
	"github.com/bouncepaw/mycorrhiza/interwiki"
	"github.com/bouncepaw/mycorrhiza/web"
	"github.com/bouncepaw/mycorrhiza/web/static"
	"github.com/bouncepaw/mycorrhiza/web/viewutil"
)

func exit() {
	process.Shutdown()
	process.Wait()
	os.Exit(1)
}

func main() {
	if err := parseCliArgs(); err != nil {
		exit()
	}

	if err := files.PrepareWikiRoot(); err != nil {
		slog.Error("Failed to prepare wiki root", "err", err)
		exit()
	}

	if err := cfg.ReadConfigFile(files.ConfigPath()); err != nil {
		slog.Error("Failed to read config", "err", err)
		exit()
	}

	if err := os.Chdir(files.HyphaeDir()); err != nil {
		slog.Error("Failed to chdir to hyphae dir",
			"err", err, "hyphaeDir", files.HyphaeDir())
		exit()
	}
	slog.Info("Running Mycorrhiza Wiki",
		"version", version.Short, "wikiDir", cfg.WikiDir)

	// Init the subsystems:
	// TODO: keep all crashes in main rather than somewhere there
	viewutil.Init()
	hyphae.Index(files.HyphaeDir())
	if err := user.InitUserDatabase(); err != nil {
		exit()
	}
	if err := history.Start(); err != nil {
		exit()
	}
	if err := history.InitGitRepo(); err != nil {
		exit()
	}
	if err := migration.Migrate(); err != nil {
		exit()
	}
	shroom.SetHeaderLinks()
	if err := categories.Init(); err != nil {
		exit()
	}
	if err := interwiki.Init(); err != nil {
		exit()
	}

	// Static files:
	static.InitFS(files.StaticFiles())

	switch {
	case !cfg.UseAuth:
		slog.Warn(
			"Authorization system is disabled. Change UseAuth to true in config.ini to enable it.",
			"ConfigPath", files.ConfigPath(),
		)
	case !user.HasAnyAdmins():
		slog.Warn("Your wiki has no admin yet. Run Mycorrhiza with -create-admin <username> option to create an admin.")
	}

	router := web.Handler()
	server := newServer(router)
	process.Go(func() {
		err := serveHTTP(server)
		if err != nil {
			process.Shutdown()
		}
	})

	signals := make(chan os.Signal)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <- signals:
		slog.Info("Received signal", "sig", sig)
		process.Shutdown()
	case <-process.Done():
	}
	timeout := 8 * time.Second
	slog.Info("Stopping HTTP server", "timeout", timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("HTTP shutdown error", "err", err)
	} else {
		slog.Info("Stopped HTTP server")
	}
	cancel()
	process.Wait()
	os.Exit(1)
}
