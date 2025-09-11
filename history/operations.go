package history

// history/operations.go
// 	Things related to writing history.
import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/bouncepaw/mycorrhiza/internal/user"
	"github.com/bouncepaw/mycorrhiza/internal/process"
	"github.com/bouncepaw/mycorrhiza/util"
)

// gitMutex is used for blocking git operations to avoid clashes.
var gitMutex = sync.RWMutex{}

// OpType is the type a history operation has. Callers shall set appropriate optypes when creating history operations.
type OpType int

const (
	// TypeNone represents an empty operation. Not to be used in practice.
	TypeNone OpType = iota
	// TypeEditText represents an edit of hypha text part.
	TypeEditText
	// TypeEditBinary represents an addition or replacement of hypha media.
	TypeEditBinary
	// TypeDeleteHypha represents a hypha deletion
	TypeDeleteHypha
	// TypeRenameHypha represents a hypha renaming
	TypeRenameHypha
	// TypeRemoveMedia represents media removal
	TypeRemoveMedia
	// TypeMarkupMigration represents a wikimind-powered automatic markup migration procedure
	TypeMarkupMigration
)

var (
	ErrOperationDone = errors.New("history operation is already done")
)

// Op is an object representing a history operation.
type Op struct {
	Type         OpType
	userMsg      string
	name         string
	email        string
	filesChanged bool
	err          error
	done         bool
}

type ReadOp struct {
}

// Operation is a constructor of a history operation.
func Operation(opType OpType) *Op {
	gitMutex.Lock()
	hop := &Op{
		Type:         opType,
		name:         "anon",
		email:        "anon@mycorrhiza",
		filesChanged: false,
		err:          nil,
		done:         false,
	}
	return hop
}

// git operation maker helper
func (hop *Op) gitop(args ...string) *Op {
	if hop.HasError() {
		return hop
	}
	if hop.done {
		return hop.withErr(ErrOperationDone)
	}
	_, err := gitsh(args...)
	if err != nil {
		return hop.withErr(err)
	}
	return hop
}

func (hop *Op) gitfileop(args []string, files ...string) *Op {
	if hop.HasError() {
		return hop
	}
	if hop.done {
		return hop.withErr(ErrOperationDone)
	}
	chunkSize := 64
	nargs := len(args)
	cmd := make([]string, nargs + chunkSize)
	copy(cmd, args)
	for chunk := range slices.Chunk(files, chunkSize) {
		hop.SetFilesChanged()
		nfiles := copy(cmd[nargs:], chunk)
		_, err := gitsh(cmd[:nargs + nfiles]...)
		if err != nil {
			return hop.withErr(err)
		}
	}
	return hop
}

// withErr appends the `err` to the list of errors.
func (hop *Op) withErr(err error) *Op {
	hop.err = err
	return hop
}

// WithErrAbort appends the `err` to the list of errors and immediately aborts the operation.
func (hop *Op) WithErrAbort(err error) *Op {
	return hop.withErr(err).Abort()
}

func (hop *Op) SetFilesChanged() *Op {
	hop.filesChanged = true
	return hop
}

// WithFilesRemoved git-rm-s all passed `paths`. Paths can be rooted or not. Paths that are empty strings are ignored.
func (hop *Op) WithFilesRemoved(paths ...string) *Op {
	paths = slices.DeleteFunc(paths, func(path string) bool {
		return path == ""
	})
	return hop.gitfileop([]string{"rm", "--"}, paths...)
}

// WithFilesRenamed git-mv-s all passed keys of `pairs` to values of `pairs`. Paths can be rooted ot not. Empty keys are ignored.
func (hop *Op) WithFilesRenamed(pairs map[string]string) *Op {
	if hop.HasError() {
		return hop
	}
	if hop.done {
		return hop.withErr(ErrOperationDone)
	}
	hop.SetFilesChanged()
	for from, to := range pairs {
		if from != "" {
			if err := os.MkdirAll(filepath.Dir(to), os.ModeDir|0770); err != nil {
				return hop.withErr(err)
			}
			hop.gitop("mv", "--force", from, to)
		}
	}
	return hop
}

// WithFiles stages all passed `paths`. Paths can be rooted or not.
func (hop *Op) WithFiles(paths ...string) *Op {
	for i, path := range paths {
		paths[i] = util.ShorterPath(path)
	}
	// 1 git operation is more effective than n operations.
	return hop.gitfileop([]string{"add"}, paths...)
}

// Apply applies history operation by doing the commit. You do not need to call Abort afterwards.
func (hop *Op) Apply() *Op {
	if hop.done {
		return hop
	}
	if hop.filesChanged {
		hop.gitop(
			"commit",
			"--author", fmt.Sprintf("%s<%s>", hop.name, hop.email),
			"-m", hop.userMsg,
			"--no-gpg-sign",
		)
	}
	if hop.HasError() {
		return hop.Abort()
	}
	gitMutex.Unlock()
	hop.done = true
	return hop
}

// Abort aborts the history operation.
func (hop *Op) Abort() *Op {
	if hop.done {
		return hop
	}
	if hop.filesChanged {
		if err := gitReset(); err != nil {
			process.Shutdown()
		}
	}
	gitMutex.Unlock()
	hop.done = true
	return hop
}

// WithMsg sets what message will be used for the future commit. If user message exceeds one line, it is stripped down.
func (hop *Op) WithMsg(userMsg string) *Op {
	i := strings.IndexAny(userMsg, "\r\n")
	if i >= 0 {
		userMsg = userMsg[:i]
	}
	hop.userMsg = userMsg
	return hop
}

// WithUser sets a user for the commit.
func (hop *Op) WithUser(u *user.User) *Op {
	u.RLock()
	if u.Group != "anon" {
		hop.name = u.Name
		hop.email = u.Name + "@mycorrhiza"
	}
	u.RUnlock()
	return hop
}

// HasErrors checks whether operation has errors appended.
func (hop *Op) HasError() bool {
	return hop.err != nil
}

// HasErrors checks whether operation has errors appended.
func (hop *Op) Err() error {
	return hop.err
}

// FirstErrorText extracts first error appended to the operation.
func (hop *Op) ErrorText() string {
	return hop.err.Error()
}

func ReadOperation() *ReadOp {
	gitMutex.RLock()
	return &ReadOp{}
}

func (hop *ReadOp) End() *ReadOp {
	gitMutex.RUnlock()
	return hop
}

