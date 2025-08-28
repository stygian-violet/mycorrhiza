package user

import (
	"fmt"
	"iter"
	"log/slog"
	"slices"
	"sort"
	"sync"
	"time"

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
func ListUsersWithGroup(group string) []string {
	var filtered []string
	for u := range YieldUsers() {
		u.RLock()
		if u.Group == group {
			filtered = append(filtered, u.Name)
		}
		u.RUnlock()
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
	for u := range YieldUsers() {
		u.RLock()
		admin := u.Group == "admin"
		u.RUnlock()
		if admin {
			return true
		}
	}
	return false
}

// CredentialsOK checks whether a correct user-password pair is provided
func CredentialsOK(username, password string) bool {
	return ByName(username).isCorrectPassword(password)
}

// ByToken finds a user by provided session token
func ByToken(token string) *User {
	tokensMutex.RLock()
	session, ok := tokens[token]
	tokensMutex.RUnlock()
	switch {
	case !ok:
		return EmptyUser()
	case session.Expired():
		slog.Info("Session expired", "data", session)
		terminateSession(token)
		return EmptyUser()
	default:
		session.Lock()
		username := session.Username
		session.LastUsed = time.Now()
		session.Unlock()
		usersMutex.RLock()
		user, ok := users[username]
		usersMutex.RUnlock()
		if !ok {
			slog.Info("Session user does not exist", "data", session)
			terminateSession(token)
			return EmptyUser()
		} else {
			sendSessionEvent(SessionActive)
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
	return EmptyUser()
}

// DeleteUser removes a user by one's name and saves user database.
func DeleteUser(name string) error {
	usersMutex.Lock()
	user, exists := users[name]
	if exists {
		delete(users, name)
	}
	usersMutex.Unlock()
	if !exists {
		return nil
	}
	user.Lock()
	user.Name = "anon"
	user.Group = "anon"
	user.Password = ""
	user.Unlock()
	sessions := 0
	tokensMutex.Lock()
	for token, session := range tokens {
		session.RLock()
		if session.Username == name {
			delete(tokens, token)
			sessions++
		}
		session.RUnlock()
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
		session.RLock()
		if session.Username == username {
			sessions = append(sessions, session)
		}
		session.RUnlock()
	}
	if uint(len(sessions)) > cfg.SessionLimit {
		slog.Info(
			"Session limit exceeded",
			"username", username, "sessions", len(sessions),
		)
		slices.SortFunc(sessions, LeastRecentlyUsedSession)
		sessions = sessions[:uint(len(sessions)) - cfg.SessionLimit]
		for _, session := range sessions {
			session.Lock()
			slog.Info("Terminating session", "data", session)
			session.Username = "anon"
			delete(tokens, session.Token)
			session.Unlock()
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
		if !exists {
			tokens[token] = session
			limitSessions(username)
		}
		tokensMutex.Unlock()
		if !exists {
			slog.Info("Added session", "username", username, "session", session)
			sendSessionEvent(SessionChanged)
			return session, nil
		}
	}
	return nil, fmt.Errorf("failed to generate a unique token after %d tries", i)
}

func terminateSession(token string) {
	tokensMutex.Lock()
	session, exists := tokens[token]
	if exists {
		delete(tokens, token)
	}
	tokensMutex.Unlock()
	if exists {
		session.Lock()
		slog.Info("Terminating session", "data", session)
		session.Username = "anon"
		session.Unlock()
		sendSessionEvent(SessionChanged)
	}
}

func UsersInGroups() (admins []string, moderators []string, editors []string, readers []string) {
	for u := range YieldUsers() {
		u.RLock()
		switch u.Group {
		// What if we place the users into sorted slices?
		case "admin":
			admins = append(admins, u.Name)
		case "moderator":
			moderators = append(moderators, u.Name)
		case "editor", "trusted":
			editors = append(editors, u.Name)
		case "reader":
			readers = append(readers, u.Name)
		}
		u.RUnlock()
	}
	sort.Strings(admins)
	sort.Strings(moderators)
	sort.Strings(editors)
	sort.Strings(readers)
	return
}
