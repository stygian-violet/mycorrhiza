package user

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"

	"golang.org/x/crypto/bcrypt"
)

// User contains information about a given user required for identification.
type User struct {
	// Name is a username. It must follow hypha naming rules.
	Name         string    `json:"name"`
	Group        string    `json:"group"`
	Password     string    `json:"hashed_password"`
	RegisteredAt time.Time `json:"registered_on"`
	// Source is where the user from. Valid values: local, telegram.
	Source string `json:"source"`
	sync.RWMutex

	// A note about why HashedPassword is string and not []byte. The reason is
	// simple: golang's json marshals []byte as slice of numbers, which is not
	// acceptable.
}

type Session struct {
	Token     string    `json: "token"`
	Username  string    `json: "username"`
	LastUsed  time.Time `json: "last_used"`
	sync.RWMutex
}

// Route — Right (more is more right)
var minimalRights = map[string]int{
	"text":                 0,
	"backlinks":            0,
	"history":              0,
	"text-search":          0,
	"media":                1,
	"edit":                 1,
	"upload-binary":        1,
	"rename":               1,
	"upload-text":          1,
	"add-to-category":      1,
	"remove-from-category": 1,
	"remove-media":         2,
	"update-header-links":  3,
	"delete":               3,
	"reindex":              4,
	"admin":                4,
	"admin/shutdown":       4,
}

var groups = []string{
	"anon",
	"reader",
	"editor",
	"trusted",
	"moderator",
	"admin",
}

// Group — Right level
var groupRight = map[string]int{
	"anon":      0,
	"reader":    0,
	"editor":    1,
	"trusted":   2,
	"moderator": 3,
	"admin":     4,
}

func setRoutePermission(route string, group string) error {
	level, ok := groupRight[group]
	if !ok {
		return fmt.Errorf("invalid group name: %s", group)
	}
	minimalRights[route] = level
	return nil
}

// ValidGroup checks whether provided user group name exists.
func ValidGroup(group string) bool {
	for _, grp := range groups {
		if grp == group {
			return true
		}
	}
	return false
}

// ValidSource checks whether provided user source name exists.
func ValidSource(source string) bool {
	return source == "local" || source == "telegram"
}

// EmptyUser constructs an anonymous user.
func EmptyUser() *User {
	return &User{
		Name:     "anon",
		Group:    "anon",
		Password: "",
		Source:   "local",
	}
}

// WikimindUser constructs the wikimind user, which is to be used for automated wiki edits and has admin privileges.
func WikimindUser() *User {
	return &User{
		Name:     "wikimind",
		Group:    "admin",
		Password: "",
		Source:   "local",
	}
}

// CanProceed checks whether user has rights to visit the provided path (and perform an action).
func (user *User) CanProceed(route string) bool {
	if !cfg.UseAuth {
		return true
	}

	user.RLock()
	defer user.RUnlock()

	right := groupRight[user.Group]
	minimalRight, specified := minimalRights[route]

	if !specified {
		return false
	}
	return right >= minimalRight
}

func (user *User) isCorrectPassword(password string) bool {
	user.RLock()
	defer user.RUnlock()
	if password == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func (user *User) IsEmpty() bool {
	user.RLock()
	res := user.Name == "anon"
	user.RUnlock()
	return res
}

// ShowLockMaybe redirects to the lock page if the user is anon and the wiki has been configured to use the lock. It returns true if the user was redirected.
func (user *User) ShowLockMaybe(w http.ResponseWriter, rq *http.Request) bool {
	user.RLock()
	lock := cfg.Locked && user.Group == "anon"
	user.RUnlock()
	if lock {
		http.Redirect(w, rq, cfg.Root + "lock", http.StatusSeeOther)
	}
	return lock
}

// Sets a new password for the user.
func (user *User) ChangePassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Lock()
	if user.Source != "local" {
		user.Unlock()
		return fmt.Errorf("Only local users can change their passwords.")
	}
	user.Password = string(hash)
	user.Unlock()
	return SaveUserDatabase()
}

func NewSession(token string, username string) *Session {
	return &Session {
		Token: token,
		Username: username,
		LastUsed: time.Now(),
	}
}

func LeastRecentlyUsedSession(a, b *Session) int {
	return a.LastUsed.Compare(b.LastUsed)
}

func MostRecentlyUsedSession(a, b *Session) int {
	return b.LastUsed.Compare(a.LastUsed)
}

func (session *Session) Expired() bool {
	session.RLock()
	defer session.RUnlock()
	now := time.Now()
	/*if now.Compare(session.LastUsed) < 0 {
		slog.Warn("Session last used in the future", "now", now, "session", session)
		return false
	}*/
	return now.Sub(session.LastUsed) > cfg.SessionTimeout
}

// IsValidUsername checks if the given username is valid.
func IsValidUsername(username string) bool {
	for _, r := range username {
		if strings.ContainsRune("?!:#@><*|\"'&%{}/", r) {
			return false
		}
	}
	return username != "anon" &&
		username != "wikimind" &&
		usernameIsWhiteListed(username)
}

func usernameIsWhiteListed(username string) bool {
	if !cfg.UseWhiteList {
		return true
	}
	for _, allowedUsername := range cfg.WhiteList {
		if allowedUsername == username {
			return true
		}
	}
	return false
}
