package user

import (
	"fmt"
	"iter"
	"log/slog"
	"slices"
	"sync"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/util"
)

var users map[string]*User
var tokens map[string]*Session

var usersMutex sync.RWMutex
var tokensMutex sync.RWMutex

func YieldUsers() iter.Seq[*User] {
	return func(yield func(*User) bool) {
		usersMutex.RLock()
		for _, user := range users {
			if !yield(user) {
				break
			}
		}
		usersMutex.RUnlock()
	}
}

// ListUsersWithGroup returns a slice with users of desired group.
func ListUsersWithPermission(permission int) []string {
	var filtered []string
	for u := range YieldUsers() {
		if u.Permission() >= permission {
			filtered = append(filtered, u.Name())
		}
	}
	return filtered
}

// Count returns total users count
func Count() (i uint64) {
	usersMutex.RLock()
	i = uint64(len(users))
	usersMutex.RUnlock()
	return i
}

func HasAnyAdmins() bool {
	p := AdminGroup().Permission()
	for u := range YieldUsers() {
		admin := u.Permission() >= p
		if admin {
			return true
		}
	}
	return false
}

// CredentialsOK checks whether a correct user-password pair is provided
func CredentialsOK(username, password string) bool {
	return ByName(username).IsCorrectPassword(password)
}

// ByToken finds a user by provided session token
func ByToken(token string) *User {
	tokensMutex.RLock()
	session, ok := tokens[token]
	tokensMutex.RUnlock()
	switch {
	case !ok:
		return emptyUser
	case session.Expired():
		slog.Info("Session expired", "data", session)
		terminateSession(token)
		return emptyUser
	default:
		user := ByName(session.Username())
		if user.IsEmpty() {
			slog.Info("Session user does not exist", "data", session)
			terminateSession(token)
			return emptyUser
		} else {
			session.Touch()
		}
		return user
	}
}

// ByName finds a user by one's username
func ByName(username string) *User {
	usersMutex.RLock()
	user, ok := users[username]
	usersMutex.RUnlock()
	if ok {
		return user
	}
	return emptyUser
}

func AddUser(user *User) error {
	usersMutex.Lock()
	users[user.Name()] = user
	usersMutex.Unlock()
	return SaveUserDatabase()
}

func ReplaceUser(old *User, new *User) error {
	oldName := old.Name()
	newName := new.Name()
	if oldName == newName {
		return AddUser(new)
	}
	usersMutex.Lock()
	delete(users, oldName)
	users[newName] = new
	usersMutex.Unlock()
	sessions := 0
	tokensMutex.Lock()
	for _, session := range tokens {
		if session.Username() == oldName {
			session.SetUsername(newName)
		}
	}
	tokensMutex.Unlock()
	if sessions > 0 {
		sendSessionEvent(SessionChanged)
	}
	return SaveUserDatabase()
}

// DeleteUser removes a user by one's name and saves user database.
func DeleteUser(name string) error {
	usersMutex.Lock()
	_, exists := users[name]
	if exists {
		delete(users, name)
	}
	usersMutex.Unlock()
	if !exists {
		return nil
	}
	sessions := 0
	tokensMutex.Lock()
	for token, session := range tokens {
		if session.Username() == name {
			delete(tokens, token)
			sessions++
		}
	}
	tokensMutex.Unlock()
	if sessions > 0 {
		sendSessionEvent(SessionChanged)
	}
	return SaveUserDatabase()
}

func limitSessions(username string) {
	if cfg.SessionLimit == 0 {
		return
	}
	var sessions []*Session
	for _, session := range tokens {
		if session.Username() == username {
			sessions = append(sessions, session)
		}
	}
	if uint(len(sessions)) > cfg.SessionLimit {
		slog.Info(
			"Session limit exceeded",
			"username", username, "sessions", len(sessions),
		)
		slices.SortFunc(sessions, LeastRecentlyUsedSession)
		sessions = sessions[:uint(len(sessions)) - cfg.SessionLimit]
		for _, session := range sessions {
			slog.Info("Terminating session", "data", session)
			session.Clear()
			delete(tokens, session.Token())
		}
	}
}

func AddSession(username string) (*Session, error) {
	i := 0
	tries := 4
	for i = 0; i < tries; i++ {
		token, err := util.RandomString(16)
		if err != nil {
			return nil, err
		}
		session := NewSession(token, username)
		tokensMutex.Lock()
		_, exists := tokens[token]
		if exists {
			tokensMutex.Unlock()
			continue
		}
		tokens[token] = session
		limitSessions(username)
		tokensMutex.Unlock()
		slog.Info("Added session", "username", username, "session", session)
		sendSessionEvent(SessionChanged)
		return session, nil
	}
	return nil, fmt.Errorf("failed to generate a unique token after %d tries", i)
}

func terminateSession(token string) {
	tokensMutex.Lock()
	session, exists := tokens[token]
	if !exists {
		tokensMutex.Unlock()
		return
	}
	delete(tokens, token)
	tokensMutex.Unlock()
	slog.Info("Terminating session", "data", session)
	session.Clear()
	sendSessionEvent(SessionChanged)
}
