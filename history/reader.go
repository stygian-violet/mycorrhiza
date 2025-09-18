package history

import (
	"net/http"
	"os"

	"github.com/bouncepaw/mycorrhiza/util"
)

type fileReader struct {}

var (
	defaultReader = fileReader{}
)

func FileReader() util.FileReadServer {
	return defaultReader
}

func (_ fileReader) ReadFile(path string) ([]byte, error) {
	// TODO: lock individual files?
	gitMutex.RLock()
	res, err := os.ReadFile(path)
	gitMutex.RUnlock()
	return res, err
}

func (_ fileReader) ServeFile(
	w http.ResponseWriter,
	rq *http.Request,
	path string,
) {
	gitMutex.RLock()
	http.ServeFile(w, rq, path)
	gitMutex.RUnlock()
}
