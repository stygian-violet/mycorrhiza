package user

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bouncepaw/mycorrhiza/internal/cfg"
	"github.com/bouncepaw/mycorrhiza/util"

	"golang.org/x/crypto/bcrypt"
)

type UserSource int

const (
	UserSourceLocal = iota
	UserSourceTelegram
)

// User contains information about a given user required for identification.
type User struct {
	name         string
	group        Group
	passwordHash []byte
	registeredAt time.Time
	source       UserSource
}

type userJson struct {
	// Name is a username. It must follow hypha naming rules.
	Name         string    `json:"name"`
	Group        string    `json:"group"`
	PasswordHash string    `json:"hashed_password"`
	RegisteredAt time.Time `json:"registered_on"`
	// Source is where the user from. Valid values: local, telegram.
	Source       string    `json:"source"`
	// A note about why HashedPassword is string and not []byte. The reason is
	// simple: golang's json marshals []byte as slice of numbers, which is not
	// acceptable.
}

var (
	// EmptyUser is an anonymous user.
	emptyUser = &User{
		name:         "anon",
		group:        EmptyGroup(),
		passwordHash: nil,
		source:       UserSourceLocal,
	}
	// WikimindUser constructs the wikimind user, which is to be used for automated wiki edits and has admin privileges.
	wikimindUser = &User{
		name:         "wikimind",
		group:        AdminGroup(),
		passwordHash: nil,
		source:       UserSourceLocal,
	}
)

func EmptyUser() *User {
	return emptyUser
}

func WikimindUser() *User {
	return wikimindUser
}

// ValidSource checks whether provided user source name exists.
func ValidSource(source string) bool {
	return source == "local" || source == "telegram"
}

func UserSourceFromString(source string) (UserSource, error) {
	switch source {
	case "local":
		return UserSourceLocal, nil
	case "telegram":
		return UserSourceTelegram, nil
	default:
		return UserSourceLocal, fmt.Errorf("invalid user source '%s'", source)
	}
}

func (user *User) String() string {
	return fmt.Sprintf("<user %s (%s)>", user.name, user.group)
}

func (user *User) MarshalJSON() ([]byte, error) {
	var src string
	switch user.source {
	case UserSourceTelegram:
		src = "telegram"
	default:
		src = "local"
	}
	return json.Marshal(userJson{
		Name:         user.name,
		Group:        user.group.Name(),
		PasswordHash: string(user.passwordHash),
		RegisteredAt: user.registeredAt,
		Source:       src,
	})
}

func (user *User) UnmarshalJSON(b []byte) error {
	var data userJson
	err := json.Unmarshal(b, &data)
	if err != nil {
		return err
	}
	var source UserSource = UserSourceLocal
	if data.Source == "telegram" {
		source = UserSourceTelegram
	}
	user.name = util.CanonicalName(data.Name)
	user.group, err = GroupByName(data.Group)
	if err != nil {
		return err
	}
	user.passwordHash = []byte(data.PasswordHash)
	user.registeredAt = data.RegisteredAt
	user.source = source
	return nil
}

func NewUser(
	name string, group string,
	password string, source string,
) (*User, error) {
	src, err := UserSourceFromString(source)
	if err != nil {
		return nil, err
	}
	grp, err := GroupByName(group)
	if err != nil {
		return nil, err
	}
	return newUserPassword(name, grp, password, time.Now(), src)
}

func newUserPassword(
	name string, group Group, password string,
	registeredAt time.Time, source UserSource,
) (*User, error) {
	hash := []byte(nil)
	if source != UserSourceTelegram {
		if password == "" {
			return nil, fmt.Errorf("password must not be empty")
		}
		var err error
		hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
	}
	return newUser(name, group, hash, registeredAt, source)
}

func newUser(
	name string, group Group, passwordHash []byte,
	registeredAt time.Time, source UserSource,
) (*User, error) {
	if !IsValidUsername(name) {
		return nil, fmt.Errorf("invalid username ‘%s’", name)
	}
	return &User{
		name:         util.CanonicalName(name),
		group:        group,
		passwordHash: passwordHash,
		registeredAt: registeredAt,
		source:       source,
	}, nil
}

func (user *User) Name() string {
	return user.name
}

func (user *User) Group() Group {
	return user.group
}

func (user *User) GroupName() string {
	return user.group.Name()
}

func (user *User) Permission() int {
	return user.group.Permission()
}

func (user *User) RegisteredAt() time.Time {
	return user.registeredAt
}

func (user *User) Source() UserSource {
	return user.source
}

func (user *User) IsLocal() bool {
	return user.source == UserSourceLocal
}

// CanProceed checks whether user has rights to visit the provided path (and perform an action).
func (user *User) CanProceed(route string) bool {
	if !cfg.UseAuth {
		return true
	}
	permission := user.group.Permission()
	required, specified := getRoutePermission(route)
	if !specified {
		return false
	}
	return permission >= required
}

func (user *User) IsCorrectPassword(password string) bool {
	if password == "" {
		return false
	}
	err := bcrypt.CompareHashAndPassword(user.passwordHash, []byte(password))
	return err == nil
}

func (user *User) IsEmpty() bool {
	return user == emptyUser
}

// ShowLock returns true if the user is anon and the wiki has been configured to use the lock.
func (user *User) ShowLock() bool {
	return cfg.Locked && user.group == emptyUser.group
}

func (user *User) WithPassword(password string) (*User, error) {
	if user.source != UserSourceLocal {
		return nil, fmt.Errorf("Only local users can change their passwords.")
	}
	return newUserPassword(
		user.name, user.group, password,
		user.registeredAt, user.source,
	)
}

func (user *User) WithGroup(group Group) (*User, error) {
	return newUser(
		user.name, group, user.passwordHash,
		user.registeredAt, user.source,
	)
}

func (user *User) WithGroupName(group string) (*User, error) {
	grp, err := GroupByName(group)
	if err != nil {
		return nil, err
	}
	return user.WithGroup(grp)
}

func (user *User) WithName(name string) (*User, error) {
	return newUser(
		name, user.group, user.passwordHash,
		user.registeredAt, user.source,
	)
}

// IsValidUsername checks if the given username is valid.
func IsValidUsername(username string) bool {
	if strings.ContainsAny(username, "?!:#@><*|\"'&%{}/") {
		return false
	}
	return username != emptyUser.name &&
		username != wikimindUser.name &&
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

func Compare(a, b *User) int {
	diff := CompareGroups(a.Group(), b.Group())
	if diff != 0 {
		return -diff
	}
	// return strings.Compare(a.Name(), b.Name())
	return a.RegisteredAt().Compare(b.RegisteredAt())
}
