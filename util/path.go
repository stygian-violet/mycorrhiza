package util

import (
	"regexp"
	"strings"

	"github.com/bouncepaw/mycorrhiza/internal/files"
)

// ShorterPath is used by handlerList to display shorter path to the files. It
// simply strips the hyphae directory name.
func ShorterPath(path string) string {
	dir := files.HyphaeDir()
	if strings.HasPrefix(path, dir) {
		return path[min(len(dir) + 1, len(path)):]
	}
	return path
}

var sanitizeExtensionRegexp = regexp.MustCompile(`[^.a-zA-Z0-9-_]+`)

func SanitizeExtension(ext string) string {
	ext = sanitizeExtensionRegexp.ReplaceAllString(ext, "")
	ext, _ = Truncate(ext, 16)
	return ext
}

// PathographicCompare compares paths preserving the path tree structure
func PathographicCompare(x string, y string) int {
	const (
		slash      int = '/'
		slashValue int = -1
	)
	// Classic lexicographical comparison with a twist
	n := min(len(x), len(y))
	for i := 0; i < n; i++ {
		// The twist: subhyphae-awareness is about pushing slash upwards
		c := int(x[i])
		if c == slash {
			c = slashValue
		}
		d := int(y[i])
		if d == slash {
			d = slashValue
		}
		diff := c - d
		if diff != 0 {
			return diff
		}
	}
	return len(x) - len(y)
}
