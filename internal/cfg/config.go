// Package cfg contains global variables that represent the current wiki
// configuration, including CLI options, configuration file values and header
// links.
package cfg

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"git.sr.ht/~bouncepaw/mycomarkup/v5/util"

	"github.com/go-ini/ini"
	"github.com/SiverPineValley/parseduration"
	"github.com/c2h5oh/datasize"
)

// These variables represent the configuration. You are not meant to modify
// them after they were set.
// See https://mycorrhiza.wiki/hypha/configuration/fields for the
// documentation.
var (
	WikiName      string
	NaviTitleIcon string

	HomeHypha           string
	UserHypha           string
	HeaderLinksHypha    string
	RedirectionCategory string

	ListenAddr        string
	URL               string
	Root              string
	CSP               string
	Referrer          string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderSize     int64
	MaxFormSize       int64
	MaxTextSize       int64
	MaxMediaSize      int64

	HTTPSEnabled bool
	CertFile     string
	KeyFile      string

	UseAuth               bool
	AllowRegistration     bool
	RegistrationLimit     uint64
	Locked                bool
	UseWhiteList          bool
	WhiteList             []string
	SessionLimit          uint
	SessionTimeout        time.Duration
	SessionUpdateInterval time.Duration
	SessionCookieDuration time.Duration

	CommonScripts []string
	ViewScripts   []string
	EditScripts   []string

	// TelegramEnabled if both TelegramBotToken and TelegramBotName are not empty strings.
	TelegramEnabled  bool
	TelegramBotToken string
	TelegramBotName  string

	FullTextSearch       FullTextSearchType
	FullTextSearchPage   bool
	FullTextLineLength   int
	FullTextLowerLimit   int
	FullTextUpperLimit   int

	GrepIgnoreMedia        bool
	GrepMatchLimitPerHypha uint
	GrepProcessLimit       uint
	GrepTimeout            time.Duration

	CustomGroups      map[string]int
	CustomPermissions map[string]string
)

// WikiDir is a full path to the wiki storage directory, which also must be a
// git repo. This variable is set in parseCliArgs().
var WikiDir string

// Config represents a Mycorrhiza wiki configuration file. This type is used
// only when reading configs.
type Config struct {
	WikiName      string `comment:"This name appears in the header and on various pages."`
	NaviTitleIcon string `comment:"This icon is used in the breadcrumbs bar."`
	Hyphae
	Network
	HTTPS
	Authorization
	Search
	Grep          `comment:"Full text search with git grep."`
	CustomScripts `comment:"You can specify additional scripts to load on different kinds of pages, delimited by a comma ',' sign."`
	Telegram      `comment:"You can enable Telegram authorization. Follow these instructions: https://core.telegram.org/widgets/login#setting-up-a-bot"`
}

// Hyphae is a section of Config which has fields related to special hyphae.
type Hyphae struct {
	HomeHypha           string `comment:"This hypha will be the main (index) page of your wiki, served on /."`
	UserHypha           string `comment:"This hypha is used as a prefix for user hyphae."`
	HeaderLinksHypha    string `comment:"You can also specify a hypha to populate your own custom header links from."`
	RedirectionCategory string `comment:"Redirection hyphae will be added to this category. Default: redirection."`
}

// Network is a section of Config that has fields related to network stuff.
type Network struct {
	ListenAddr        string
	URL               string `comment:"Set your wiki's public URL here. It's used for OpenGraph generation and syndication feeds."`
	Root              string `comment:"Set your wiki's root path here."`
	CSP               string `comment:"Content-Security-Policy header."`
	Referrer          string `comment:"Referrer-Policy header."`
	ReadHeaderTimeout string `comment:"The amount of time allowed to read request headers. If zero, the value of ReadTimeout is used. If negative, or if zero and ReadTimeout is zero or negative, there is no timeout."`
	ReadTimeout       string `comment:"Maximum duration for reading the entire request, including the body. A zero or negative value means there will be no timeout."`
	WriteTimeout      string `comment:"Maximum duration before timing out writes of the response. A zero or negative value means there will be no timeout."`
	IdleTimeout       string `comment:"Maximum amount of time to wait for the next request when keep-alives are enabled. If zero, the value of ReadTimeout is used. If negative, or if zero and ReadTimeout is zero or negative, there is no timeout."`
	MaxHeaderSize     string `comment:"Maximum size of the request headers, including the request line. If zero, the default limit (1MB) is used."`
	MaxFormSize       string `comment:"Maximum size of a post form without files. If zero, the default limit (10MB) is used."`
	MaxTextSize       string `comment:"Maximum size of a hypha's text. If zero, there is no limit."`
	MaxMediaSize      string `comment:"Maximum size of a media file. If zero, there is no limit."`
}

type HTTPS struct {
	HTTPSEnabled bool   `comment:"Whether to enable HTTPS. If true, CertFile and KeyFile should be set."`
	CertFile     string `comment:"Certificate file. If the certificate is signed by a certificate authority, the file should be the concatenation of the server's certificate, any intermediates, and the CA's certificate."`
	KeyFile      string `comment:"Private key file."`
}

// CustomScripts is a section with paths to JavaScript files that are loaded on
// specified pages.
type CustomScripts struct {
	// CommonScripts: everywhere...
	CommonScripts []string `delim:"," comment:"These scripts are loaded from anywhere."`
	// ViewScripts: /hypha, /rev
	ViewScripts []string `delim:"," comment:"These scripts are only loaded on view pages."`
	// Edit: /edit
	EditScripts []string `delim:"," comment:"These scripts are only loaded on the edit page."`
}

// Authorization is a section of Config that has fields related to
// authorization and authentication.
type Authorization struct {
	UseAuth               bool
	AllowRegistration     bool
	RegistrationLimit     uint64   `comment:"This field controls the maximum amount of allowed registrations."`
	Locked                bool     `comment:"Set if users have to authorize to see anything on the wiki."`
	UseWhiteList          bool     `comment:"If true, WhiteList is used. Else it is not used."`
	WhiteList             []string `delim:"," comment:"Usernames of people who can log in to your wiki separated by comma."`
	SessionLimit          uint     `comment:"Maximum number of login sessions per user. If exceeded, the least recently used session is terminated. If the number is zero, there is no limit."`
	SessionTimeout        string   `comment:"Maximum period of inactivity before a session is terminated."`
	SessionUpdateInterval string   `comment:"How often session activity time is saved."`
	SessionCookieDuration string   `comment:"How long session cookies last."`
	// TODO: let admins enable auth-less editing
}

// Telegram is the section of Config that sets Telegram authorization.
type Telegram struct {
	TelegramBotToken string `comment:"Token of your bot."`
	TelegramBotName  string `comment:"Username of your bot, sans @."`
}

type Search struct {
	FullText             string `comment:"Full text search type. Options: none, grep"`
	FullTextLineLength   int   `comment:"Maximum length of a single line of a full text search result. If the number is zero, only hypha links are shown. If the number is negative, there is no limit."`
	FullTextLowerLimit   int    `comment:"Maximum number of full text search results shown in the /title-search/ page. If the number is zero, full text search is disabled for the page. If the number is negative, there is no limit."`
	FullTextUpperLimit   int    `comment:"Maximum number of search results shown in the /text-search/ page. If the number is zero, the page does not exist. If the number is negative, there is no limit."`
}

type Grep struct {
	GrepIgnoreMedia        bool   `comment:"Whether to exclude non-binary media files from full text search"`
	GrepMatchLimitPerHypha uint   `comment:"Maximum number of matched lines per hypha. If the number is zero, there is no limit."`
	GrepProcessLimit       uint   `comment:"Maximum number of parallel grep processes. If exceeded, full text search returns an error. If the number is zero, there is no limit."`
	GrepTimeout            string `comment:"Maximum execution time of grep processes. If the duration is zero, there is no limit."`
}

type FullTextSearchType int

const (
	FullTextDisabled FullTextSearchType = iota
	FullTextGrep
	// TODO: FullTextIndex ?
)

func FullTextSearchTypeFromString(value string) (FullTextSearchType, error) {
	value = strings.ToLower(value)
	switch value {
	case "none", "off", "false", "disabled":
		return FullTextDisabled, nil
	case "grep":
		return FullTextGrep, nil
	default:
		return FullTextDisabled, fmt.Errorf("invalid full text search type: %s", value)
	}
}

func (t FullTextSearchType) String() string {
	switch t {
	case FullTextDisabled:
		return "none"
	case FullTextGrep:
		return "grep"
	default:
		return "none"
	}
}

func pd(value string, key string) (time.Duration, error) {
	if value == "0" {
		return time.Duration(0), nil
	}
	res, err := parseduration.ParseDuration(value)
	if err != nil {
		err = fmt.Errorf("failed to parse %s: %s: %s", key, value, err.Error())
	}
	return res, err
}

func ps(value string, key string) (int64, error) {
	res, err := datasize.ParseString(value)
	if err != nil {
		err = fmt.Errorf("failed to parse %s: %s: %s", key, value, err.Error())
	}
	return int64(res), err
}

func intKeysHash(s *ini.Section) (map[string]int, error) {
	res := make(map[string]int)
	keys := s.Keys()
	for _, k := range keys {
		v, err := k.Int()
		if err != nil {
			return nil, fmt.Errorf(
				"failed to get key %s of section %s: %s",
				k.Name(), s.Name(), err.Error(),
			)
		}
		res[k.Name()] = v
	}
	return res, nil
}

// ReadConfigFile reads a config on the given path and stores the
// configuration. Call it sometime during the initialization.
func ReadConfigFile(path string) error {
	cfg := &Config{
		WikiName:      "Mycorrhiza Wiki",
		NaviTitleIcon: "🍄",
		Hyphae: Hyphae{
			HomeHypha:           "home",
			UserHypha:           "u",
			HeaderLinksHypha:    "",
			RedirectionCategory: "redirection",
		},
		Network: Network{
			ListenAddr:        "127.0.0.1:1737",
			URL:               "",
			Root:              "/",
			CSP:               "default-src 'self' telegram.org *.telegram.org; "+
			                   "img-src * data:; media-src *; style-src *; font-src * data:",
			Referrer:          "no-referrer",
			ReadHeaderTimeout: "0",
			ReadTimeout:       "5m",
			WriteTimeout:      "5m",
			IdleTimeout:       "0",
			MaxHeaderSize:     "1MB",
			MaxFormSize:       "10MB",
			MaxTextSize:       "0",
			MaxMediaSize:      "0",
		},
		HTTPS: HTTPS{
			HTTPSEnabled: false,
			CertFile:     "",
			KeyFile:      "",
		},
		Authorization: Authorization{
			UseAuth:               false,
			AllowRegistration:     false,
			RegistrationLimit:     0,
			Locked:                false,
			UseWhiteList:          false,
			WhiteList:             []string{},
			SessionLimit:          0,
			SessionTimeout:        "1y",
			SessionUpdateInterval: "1d",
			SessionCookieDuration: "1y",
		},
		Search: Search{
			FullText:             "grep",
			FullTextLineLength:   256,
			FullTextLowerLimit:   0,
			FullTextUpperLimit:   256,
		},
		Grep: Grep{
			GrepIgnoreMedia:        true,
			GrepMatchLimitPerHypha: 1,
			GrepProcessLimit:       16,
			GrepTimeout:            "10s",
		},
		CustomScripts: CustomScripts{
			CommonScripts: []string{},
			ViewScripts:   []string{},
			EditScripts:   []string{},
		},
		Telegram: Telegram{
			TelegramBotToken: "",
			TelegramBotName:  "",
		},
	}

	f, err := ini.Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			f = ini.Empty()

			// Save the default configuration
			err = f.ReflectFrom(cfg)
			if err != nil {
				return fmt.Errorf("Failed to serialize the config: %w", err)
			}

			// Disable key-value auto-aligning, but retain spaces around '=' sign
			ini.PrettyFormat = false
			ini.PrettyEqual = true
			if err = f.SaveTo(path); err != nil {
				return fmt.Errorf("Failed to save the config file: %w", err)
			}
		} else {
			return fmt.Errorf("Failed to open the config file: %w", err)
		}
	}

	// Map the config file to the config struct. It'll do nothing if the file
	// doesn't exist or is empty.
	if err := f.MapTo(cfg); err != nil {
		return err
	}

	configDir := filepath.Dir(path)

	// Map the struct to the global variables
	WikiName = cfg.WikiName
	NaviTitleIcon = cfg.NaviTitleIcon
	HomeHypha = util.CanonicalName(filepath.ToSlash(cfg.HomeHypha))
	UserHypha = util.CanonicalName(filepath.ToSlash(cfg.UserHypha))
	HeaderLinksHypha = util.CanonicalName(filepath.ToSlash(cfg.HeaderLinksHypha))
	RedirectionCategory = cfg.RedirectionCategory
	if ListenAddr == "" {
		ListenAddr = cfg.ListenAddr
	}
	URL = cfg.URL
	Root = filepath.ToSlash(cfg.Root)
	CSP = cfg.CSP
	Referrer = cfg.Referrer
	if ReadHeaderTimeout, err = pd(cfg.ReadHeaderTimeout, "ReadHeaderTimeout"); err != nil {
		return err
	}
	if ReadTimeout, err = pd(cfg.ReadTimeout, "ReadTimeout"); err != nil {
		return err
	}
	if WriteTimeout, err = pd(cfg.WriteTimeout, "WriteTimeout"); err != nil {
		return err
	}
	if IdleTimeout, err = pd(cfg.IdleTimeout, "IdleTimeout"); err != nil {
		return err
	}
	if MaxHeaderSize, err = ps(cfg.MaxHeaderSize, "MaxHeaderSize"); err != nil {
		return err
	}
	if MaxHeaderSize <= 0 {
		MaxHeaderSize = 1 << 20
	}
	if MaxFormSize, err = ps(cfg.MaxFormSize, "MaxFormSize"); err != nil {
		return err
	}
	if MaxFormSize <= 0 {
		MaxFormSize = 10 << 20
	}
	if MaxTextSize, err = ps(cfg.MaxTextSize, "MaxTextSize"); err != nil {
		return err
	}
	if MaxMediaSize, err = ps(cfg.MaxMediaSize, "MaxMediaSize"); err != nil {
		return err
	}
	HTTPSEnabled = cfg.HTTPSEnabled
	CertFile = cfg.CertFile
	KeyFile = cfg.KeyFile
	if HTTPSEnabled {
		switch {
		case CertFile == "":
			return errors.New("HTTPS is enabled but CertFile is not set")
		case !filepath.IsAbs(CertFile):
			CertFile = filepath.Join(configDir, CertFile)
		}
		switch {
		case KeyFile == "":
			return errors.New("HTTPS is enabled but KeyFile is not set")
		case !filepath.IsAbs(KeyFile):
			KeyFile = filepath.Join(configDir, KeyFile)
		}
	}
	UseAuth = cfg.UseAuth
	AllowRegistration = cfg.AllowRegistration
	RegistrationLimit = cfg.RegistrationLimit
	Locked = cfg.Locked
	if Locked && !UseAuth {
		slog.Warn("Makes no sense to have the lock but no auth")
		slog.Warn("Disabling lock")
		Locked = false
	}
	UseWhiteList = cfg.UseWhiteList
	WhiteList = cfg.WhiteList
	SessionLimit = cfg.SessionLimit
	if SessionTimeout, err = pd(cfg.SessionTimeout, "SessionTimeout"); err != nil {
		return err
	}
	if SessionUpdateInterval, err = pd(cfg.SessionUpdateInterval, "SessionUpdateInterval"); err != nil {
		return err
	}
	if SessionCookieDuration, err = pd(cfg.SessionCookieDuration, "SessionCookieDuration"); err != nil {
		return err
	}
	if FullTextSearch, err = FullTextSearchTypeFromString(cfg.FullText); err != nil {
		return err
	}
	FullTextLineLength = cfg.FullTextLineLength
	if FullTextLineLength == 0 {
		cfg.GrepMatchLimitPerHypha = 1
	}
	FullTextLowerLimit = cfg.FullTextLowerLimit
	FullTextUpperLimit = cfg.FullTextUpperLimit
	FullTextSearchPage = FullTextSearch != FullTextDisabled && FullTextUpperLimit != 0
	GrepIgnoreMedia = cfg.GrepIgnoreMedia
	GrepMatchLimitPerHypha = cfg.GrepMatchLimitPerHypha
	GrepProcessLimit = cfg.GrepProcessLimit
	if GrepTimeout, err = pd(cfg.GrepTimeout, "GrepTimeout"); err != nil {
		return err
	}
	CommonScripts = cfg.CommonScripts
	ViewScripts = cfg.ViewScripts
	EditScripts = cfg.EditScripts
	TelegramBotToken = cfg.TelegramBotToken
	TelegramBotName = cfg.TelegramBotName
	TelegramEnabled = (TelegramBotToken != "") && (TelegramBotName != "")

	s, err := f.GetSection("Groups")
	if err == nil {
		CustomGroups, err = intKeysHash(s)
		if err != nil {
			return err
		}
	}

	s, err = f.GetSection("Permissions")
	if err == nil {
		CustomPermissions = s.KeysHash()
	}

	if !strings.HasSuffix(Root, "/") {
		Root = Root + "/"
	}

	// This URL makes much more sense. If no URL is set or the protocol is forgotten, assume HTTP.
	if URL == "" {
		URL = "http://" + ListenAddr + cfg.Root
	} else if !strings.Contains(URL, ":") {
		URL = "http://" + URL
	}

	return nil
}
