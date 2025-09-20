package main

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
)

func newServer(handler http.Handler) *http.Server {
	return &http.Server{
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    int(cfg.MaxHeaderSize),
		Handler:           handler,
	}
}

func serveHTTP(server *http.Server) (err error) {
	if strings.HasPrefix(cfg.ListenAddr, "/") {
		err = startUnixSocketServer(server, cfg.ListenAddr)
	} else {
		server.Addr = cfg.ListenAddr
		err = startHTTPServer(server)
	}
	return err
}

func startUnixSocketServer(server *http.Server, socketPath string) error {
	err := os.Remove(socketPath)
	if err != nil {
		slog.Warn("Failed to clean up old socket", "err", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		slog.Error("Failed to start the server", "err", err)
		return err
	}
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)

	if err := os.Chmod(socketPath, 0666); err != nil {
		slog.Error("Failed to set socket permissions", "err", err)
		return err
	}

	slog.Info("Listening Unix socket", "addr", socketPath)

	if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start the server", "err", err)
		return err
	}

	return nil
}

func startHTTPServer(server *http.Server) error {
	err := error(nil)
	if cfg.HTTPSEnabled {
		slog.Info("Listening over HTTPS", "addr", server.Addr)
		err = server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile)
	} else {
		slog.Info("Listening over HTTP", "addr", server.Addr)
		err = server.ListenAndServe()
	}
	if !errors.Is(err, http.ErrServerClosed) {
		slog.Error("Failed to start the server", "err", err)
		return err
	}
	return nil
}
