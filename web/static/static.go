package static

import (
	"embed"
	"errors"
	"io/fs"
	// "log/slog"
	"os"
)

//go:embed *.css *.js *.txt icon help
var embedFS embed.FS

// FS serves all static files.
var FS HybridFS

var ErrNotInitialized = errors.New("static fs is not initialized")

// HybridFS is a filesystem that implements fs.FS. It can serve files
// from multiple filesystems, falling back on failures.
type HybridFS struct {
	fs []fs.FS
}

// Open tries to open the requested file using all filesystems provided.
// If neither succeeds, it returns the last error.
func (f HybridFS) Open(name string) (fs.File, error) {
	var file fs.File
	var err error

	for _, candidate := range f.fs {
		file, err = candidate.Open(name)
		// slog.Info("open", "name", name, "err", err)
		if err == nil {
			return file, nil
		}
	}

	if err == nil {
		err = ErrNotInitialized
	}
	return nil, err
}

// InitFS initializes the global HybridFS singleton with the wiki's own static
// files directory as a primary filesystem and the embedded one as a fallback.
func InitFS(localPath string) {
	FS = HybridFS{
		fs: []fs.FS{
			os.DirFS(localPath),
			embedFS,
		},
	}
}
