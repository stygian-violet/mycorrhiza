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
	"github.com/bouncepaw/mycorrhiza/internal/process"
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
func InitUserDatabase() error {
	if !cfg.UseAuth {
		return nil
	}
	if err := initPermissions(); err != nil {
		slog.Error("Failed to initialize permissions", "err", err)
		return err
	}
	if err := ReadUsersFromFilesystem(); err != nil {
		return err
	}
	process.Go(runSessionUpdater)
	return nil
}

// ReadUsersFromFilesystem reads all user information from filesystem and
// stores it internally.
func ReadUsersFromFilesystem() error {
	users, err := usersFromFile()
	if err != nil {
		return err
	}
	rememberUsers(users)
	return readTokensToUsers()
}

func runSessionUpdater() {
	slog.Info("Starting session updater")
	ticker := time.NewTicker(cfg.SessionUpdateInterval)
	defer ticker.Stop()
	save := false
L:
	for {
		write := false
		select {
		case <-process.Done():
			break L
		case ev, ok := <- sessionEvents:
			switch {
			case !ok:
				slog.Info("Session event channel closed")
				break L
			case ev == SessionActive:
				save = true
			case ev == SessionChanged:
				write = true
			default:
				slog.Warn("Invalid session event", "ev", ev)
			}
		case <- ticker.C:
			if save {
				slog.Info("Saving session activity")
				write = true
			}
		}
		if write {
			err := writeTokens()
			if err == nil {
				save = false
			}
		}
	}
	slog.Info("Stopping session updater")
	if save {
		slog.Info("Saving sessions")
		writeTokens()
	}
}

func sendSessionEvent(ev SessionEvent) {
	select {
	case <-process.Done():
	case sessionEvents <- ev:
	}
}

func usersFromFile() ([]*User, error) {
	var users []*User

	userFileMutex.Lock()
	contents, err := os.ReadFile(files.UserCredentialsJSON())
	userFileMutex.Unlock()
	if errors.Is(err, os.ErrNotExist) {
		return users, nil
	}
	if err != nil {
		slog.Error("Failed to read users.json", "err", err)
		return users, err
	}

	err = json.Unmarshal(contents, &users)
	if err != nil {
		slog.Error("Failed to unmarshal users.json contents", "err", err)
		return users, err
	}

	slog.Info("Indexed users", "n", len(users))
	return users, nil
}

func rememberUsers(userList []*User) {
	newUsers := make(map[string]*User, len(userList))
	for _, user := range userList {
		name := user.Name()
		if !IsValidGroup(user.Group()) {
			slog.Warn("Invalid user group", "user", user)
			u, err := user.WithGroup(EmptyUser.Group())
			if err != nil {
				slog.Error("Failed to change group", "user", user, "err", err)
			} else {
				user = u
			}
		}
		if IsValidUsername(name) {
			user2, exists := newUsers[name]
			if exists {
				slog.Error("User already exists", "new", user, "existing", user2)
			} else {
				newUsers[name] = user
			}
		} else {
			slog.Warn("Invalid username", "user", user)
		}
	}
	usersMutex.Lock()
	users = newUsers
	usersMutex.Unlock()
}

func readTokensToUsers() error {
	contents, err := os.ReadFile(files.TokensJSON())
	if os.IsNotExist(err) {
		tokensMutex.Lock()
		tokens = make(map[string]*Session)
		tokensMutex.Unlock()
		return nil
	}
	if err != nil {
		slog.Error("Failed to read tokens.json", "err", err)
		return err
	}

	var sessions []*Session
	userSessions := make(map[string]uint)
	err = json.Unmarshal(contents, &sessions)
	if err != nil {
		slog.Error("Failed to unmarshal tokens.json contents", "err", err)
		tokensMutex.Lock()
		tokens = make(map[string]*Session)
		tokensMutex.Unlock()
		return nil
	}
	slices.SortFunc(sessions, MostRecentlyUsedSession)

	active := 0
	invalid := 0

	newTokens := make(map[string]*Session, len(sessions))
	for _, session := range sessions {
		switch {
		case session.Expired():
			slog.Info("Session expired", "session", session)
			invalid++
		case (cfg.SessionLimit > 0 &&
			userSessions[session.Username()] == cfg.SessionLimit):
			slog.Info(
				"Session limit exceeded",
				"user", session.Username,
				"limit", cfg.SessionLimit,
				"session", session,
			)
			invalid++
		default:
			active++
			userSessions[session.Username()]++
			newTokens[session.Token()] = session
		}
	}
	tokensMutex.Lock()
	tokens = newTokens
	tokensMutex.Unlock()
	slog.Info("Indexed sessions", "active", active, "invalid", invalid)
	return nil
}

// SaveUserDatabase stores current user credentials into JSON file by configured path.
func SaveUserDatabase() error {
	return writeUserCredentials()
}

func writeUserCredentials() error {
	userFileMutex.Lock()
	defer userFileMutex.Unlock()

	var userList []*User
	for u := range YieldUsers() {
		userList = append(userList, u)
	}
	blob, err := json.MarshalIndent(userList, "", "\t")
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

func writeTokens() error {
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

	blob, err := json.MarshalIndent(sessionList, "", "\t")
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
