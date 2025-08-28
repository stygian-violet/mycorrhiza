package user

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/internal/files"
	"github.com/bouncepaw/mycorrhiza/util"
)

type SessionEvent int
const (
	SessionActive SessionEvent = iota
	SessionChanged
)

var (
	userFileMutex sync.Mutex
	sessionEvents = make(chan SessionEvent, 16)
)

// InitUserDatabase loads users, if necessary. Call it during initialization.
func InitUserDatabase() {
	if !cfg.UseAuth {
		return
	}
	ReadUsersFromFilesystem()
	go runSessionUpdater()
}

// ReadUsersFromFilesystem reads all user information from filesystem and
// stores it internally.
func ReadUsersFromFilesystem() {
	rememberUsers(usersFromFile())
	readTokensToUsers()
}

func runSessionUpdater() {
	ticker := time.NewTicker(cfg.SessionUpdateInterval)
	defer ticker.Stop()
	save := false
	for {
		select {
		case ev, ok := <- sessionEvents:
			switch {
			case !ok:
				slog.Info("Session event channel closed")
				slog.Info("Saving sessions")
				dumpTokens()
				return
			case ev == SessionActive:
				save = true
			case ev == SessionChanged:
				err := dumpTokens()
				if err == nil {
					save = false
				}
			default:
				slog.Warn("Invalid session event", "ev", ev)
			}
		case <- ticker.C:
			if save {
				slog.Info("Saving session activity")
				err := dumpTokens()
				if err == nil {
					save = false
				}
			}
		}
	}
}

func usersFromFile() []*User {
	var users []*User

	userFileMutex.Lock()
	contents, err := os.ReadFile(files.UserCredentialsJSON())
	userFileMutex.Unlock()
	if errors.Is(err, os.ErrNotExist) {
		return users
	}
	if err != nil {
		slog.Error("Failed to read users.json", "err", err)
		os.Exit(1)
	}

	err = json.Unmarshal(contents, &users)
	if err != nil {
		slog.Error("Failed to unmarshal users.json contents", "err", err)
		os.Exit(1)
	}

	for _, u := range users {
		u.Name = util.CanonicalName(u.Name)
		if u.Source == "" {
			u.Source = "local"
		}
	}
	slog.Info("Indexed users", "n", len(users))
	return users
}

func rememberUsers(userList []*User) {
	usersMutex.Lock()
	users = make(map[string]*User, len(userList))
	for _, user := range userList {
		if IsValidUsername(user.Name) {
			user2, exists := users[user.Name]
			if exists {
				slog.Error("User already exists", "new", user, "existing", user2)
			} else {
				users[user.Name] = user
			}
		}
	}
	usersMutex.Unlock()
}

func readTokensToUsers() {
	contents, err := os.ReadFile(files.TokensJSON())
	if os.IsNotExist(err) {
		tokensMutex.Lock()
		tokens = make(map[string]*Session)
		tokensMutex.Unlock()
		return
	}
	if err != nil {
		slog.Error("Failed to read tokens.json", "err", err)
		os.Exit(1)
	}

	var sessions []*Session
	userSessions := make(map[string]uint)
	err = json.Unmarshal(contents, &sessions)
	if err != nil {
		slog.Error("Failed to unmarshal tokens.json contents", "err", err)
	}
	slices.SortFunc(sessions, MostRecentlyUsedSession)

	active := 0
	invalid := 0
	tokensMutex.Lock()
	tokens = make(map[string]*Session, len(sessions))
	for _, session := range sessions {
		switch {
		case session.Expired():
			slog.Info("Session expired", "session", session)
			invalid++
		case cfg.SessionLimit > 0 && userSessions[session.Username] == cfg.SessionLimit:
			slog.Info(
				"Session limit exceeded",
				"user", session.Username,
				"limit", cfg.SessionLimit,
				"session", session,
			)
			invalid++
		default:
			active++
			userSessions[session.Username]++
			tokens[session.Token] = session
		}
	}
	tokensMutex.Unlock()
	slog.Info("Indexed sessions", "active", active, "invalid", invalid)
}

// SaveUserDatabase stores current user credentials into JSON file by configured path.
func SaveUserDatabase() error {
	return dumpUserCredentials()
}

func dumpUserCredentials() error {
	userFileMutex.Lock()
	defer userFileMutex.Unlock()

	var userList []*User
	for u := range YieldUsers() {
		userList = append(userList, u)
	}
	for _, u := range userList {
		u.RLock()
	}
	blob, err := json.MarshalIndent(userList, "", "\t")
	for _, u := range userList {
		u.RUnlock()
	}
	if err != nil {
		slog.Error("Failed to marshal users.json", "err", err)
		return err
	}

	err = os.WriteFile(files.UserCredentialsJSON(), blob, 0660)
	if err != nil {
		slog.Error("Failed to write users.json", "err", err)
		return err
	}

	return nil
}

func dumpTokens() error {
	var sessionList []*Session
	tokensMutex.Lock()
	for token, session := range tokens {
		if session.Expired() {
			slog.Info("Session expired", "data", session)
			delete(tokens, token)
		} else {
			sessionList = append(sessionList, session)
		}
	}
	tokensMutex.Unlock()

	for _, session := range sessionList {
		session.RLock()
	}
	blob, err := json.MarshalIndent(sessionList, "", "\t")
	for _, session := range sessionList {
		session.RUnlock()
	}
	if err != nil {
		slog.Error("Failed to marshal tokens.json", "err", err)
		return err
	}

	err = os.WriteFile(files.TokensJSON(), blob, 0660)
	if err != nil {
		slog.Error("Failed to write tokens.json", "err", err)
		return err
	}
	slog.Info("Saved sessions", "n", len(sessionList))

	return nil
}
