package user

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
)

type Session struct {
	token     string
	username  string
	lastUsed  time.Time
	mutex     sync.RWMutex
}

type sessionJson struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	LastUsed  time.Time `json:"last_used"`
}

func (session *Session) String() string {
	session.mutex.RLock()
	res := fmt.Sprintf(
		"<session %s of user %s>",
		session.token, session.username,
	)
	session.mutex.RUnlock()
	return res
}

func (session *Session) MarshalJSON() ([]byte, error) {
	session.mutex.RLock()
	data := sessionJson{
		Token:    session.token,
		Username: session.username,
		LastUsed: session.lastUsed,
	}
	session.mutex.RUnlock()
	return json.Marshal(data)
}

func (session *Session) UnmarshalJSON(b []byte) error {
	var data sessionJson
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	session.token = data.Token
	session.username = data.Username
	session.lastUsed = data.LastUsed
	return nil
}

func NewSession(token string, username string) *Session {
	return newSession(token, username, time.Now())
}

func newSession(token string, username string, lastUsed time.Time) *Session {
	return &Session {
		token: token,
		username: username,
		lastUsed: lastUsed,
	}
}

func (session *Session) Token() string {
	return session.token
}

func (session *Session) Username() string {
	session.mutex.RLock()
	res := session.username
	session.mutex.RUnlock()
	return res
}

func (session *Session) SetUsername(username string) {
	session.mutex.Lock()
	session.username = username
	session.mutex.Unlock()
}

func (session *Session) Clear() {
	session.mutex.Lock()
	session.username = emptyUser.Name()
	session.mutex.Unlock()
}

func (session *Session) LastUsed() time.Time {
	session.mutex.RLock()
	res := session.lastUsed
	session.mutex.RUnlock()
	return res
}

func (session *Session) Expired() bool {
	lastUsed := session.LastUsed()
	now := time.Now()
	/*if now.Compare(session.LastUsed) < 0 {
		slog.Warn("Session last used in the future", "now", now, "session", session)
		return false
	}*/
	return now.Sub(lastUsed) > cfg.SessionTimeout
}

func (session *Session) Touch() {
	session.mutex.Lock()
	session.lastUsed = time.Now()
	session.mutex.Unlock()
	sendSessionEvent(SessionActive)
}

func LeastRecentlyUsedSession(a, b *Session) int {
	return a.LastUsed().Compare(b.LastUsed())
}

func MostRecentlyUsedSession(a, b *Session) int {
	return b.LastUsed().Compare(a.LastUsed())
}
