// config.go

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	rf "github.com/meinside/rss-feeds-go"
)

const (
	defaultGoogleAIModel   = `gemini-3.6-flash`
	defaultDesiredLanguage = `English`

	defaultFetchFeedsIntervalSeconds = 60 * 3 // = 3 minutes
	defaultFetchFeedsTimeoutSeconds  = 60 * 1 // = 1 minute

	defaultExaoneKeyFilename = `exaone-key.json` // in `db_files_dir`
)

// config struct
type config struct {
	// System
	GoogleAIAPIKey  *string  `json:"google_ai_api_key,omitempty"`
	GoogleAIAPIKeys []string `json:"google_ai_api_keys,omitempty"`
	GoogleAIModels  []string `json:"google_ai_models,omitempty"`
	// EXAONE account for issuing and renewing an api key automatically (keys
	// expire in 30 days, so static keys are not supported).
	//
	// EXAONE is round-robined with Gemini for text contents only: without
	// Gemini, files, YouTube videos, and unfetchable urls are cached with a
	// failed summary (see `drop_items_with_failed_summaries`)
	ExaoneAccount    *configExaoneAccount `json:"exaone_account,omitempty"`
	ExaoneModels     []string             `json:"exaone_models,omitempty"` // library default when empty
	DBFilesDirectory string               `json:"db_files_dir"`
	DesiredLanguage  *string              `json:"desired_language,omitempty"`
	Verbose          bool     `json:"verbose,omitempty"`

	// RSS
	RSSFeeds                  []configRSSFeed `json:"rss_feeds"`
	FetchFeedsIntervalSeconds int             `json:"fetch_feeds_interval_seconds,omitempty"`
	FetchFeedsTimeoutSeconds  int             `json:"fetch_feeds_timeout_seconds,omitempty"`
	PermittedUserAgents       []string        `json:"permitted_user_agents,omitempty"`

	// RSS server port
	RSSServerPort int `json:"rss_server_port"`
}

// configRSSFeed struct
type configRSSFeed struct {
	Name          string   `json:"name"`
	CacheFilename string   `json:"cache_filename"`
	ServePath     string   `json:"serve_path"`
	FeedURLs      []string `json:"feed_urls"`

	PublishTitle       *string `json:"publish_title,omitempty"`
	PublishLink        *string `json:"publish_link,omitempty"`
	PublishDescription *string `json:"publish_description,omitempty"`
	PublishAuthor      *string `json:"publish_author,omitempty"`
	PublishEmail       *string `json:"publish_email,omitempty"`

	DropItemsWithFailedSummaries bool `json:"drop_items_with_failed_summaries,omitempty"`
}

// configExaoneAccount struct
//
// NOTE: the issued key replaces the account's previous one, so the account
// must not be used by other machines (or with a different `key_file`).
type configExaoneAccount struct {
	Email    string `json:"email"`
	Password string `json:"password"`

	// KeyFile keeps the issued key across restarts, as the key cannot be
	// retrieved again: without it, every start would issue a new key
	// (default: `exaone-key.json` in `db_files_dir`)
	KeyFile string `json:"key_file,omitempty"`
}

// redacted returns a copy of the config with api keys and passwords masked,
// for logging.
func (c config) redacted() config {
	if c.GoogleAIAPIKey != nil {
		c.GoogleAIAPIKey = new(maskAPIKey(*c.GoogleAIAPIKey))
	}
	c.GoogleAIAPIKeys = maskAPIKeys(c.GoogleAIAPIKeys)
	if c.ExaoneAccount != nil {
		account := *c.ExaoneAccount
		account.Password = maskedValue
		c.ExaoneAccount = &account
	}
	return c
}

// maskAPIKeys returns a new slice of masked api keys (nil for nil).
func maskAPIKeys(keys []string) []string {
	if keys == nil {
		return nil
	}
	masked := make([]string, len(keys))
	for i, key := range keys {
		masked[i] = maskAPIKey(key)
	}
	return masked
}

// maskAPIKey masks given api key, leaving only its last few characters for
// telling keys apart.
//
// NOTE: short keys are masked entirely, as their last characters would be a
// large part of them.
func maskAPIKey(key string) string {
	const (
		visibleChars     = 4
		minLenForVisible = 16
	)
	if len(key) < minLenForVisible {
		return maskedValue
	}
	return maskedValue + key[len(key)-visibleChars:]
}

// maskedValue replaces secrets in logs
const maskedValue = "********"

// read config from given `configFilepath`
func readConfig(configFilepath string) (conf config, err error) {
	var bytes []byte
	if bytes, err = os.ReadFile(configFilepath); err == nil {
		if bytes, err = rf.StandardizeJSON(bytes); err == nil {
			if err = json.Unmarshal(bytes, &conf); err == nil {
				// validate required fields
				if conf.GoogleAIAPIKey == nil && len(conf.GoogleAIAPIKeys) == 0 && conf.ExaoneAccount == nil {
					return conf, fmt.Errorf("at least one of 'google_ai_api_key', 'google_ai_api_keys', or 'exaone_account' is required")
				}
				// NOTE: checked here, as a login with them would count toward the account lock
				if conf.ExaoneAccount != nil && (conf.ExaoneAccount.Email == "" || conf.ExaoneAccount.Password == "") {
					return conf, fmt.Errorf("'exaone_account' requires 'email' and 'password'")
				}
				if conf.DBFilesDirectory == "" {
					return conf, fmt.Errorf("'db_files_dir' is required")
				}
				if len(conf.RSSFeeds) == 0 {
					return conf, fmt.Errorf("'rss_feeds' must contain at least one entry")
				}
				if conf.RSSServerPort <= 0 {
					return conf, fmt.Errorf("'rss_server_port' must be a positive number")
				}

				// set default values
				if len(conf.GoogleAIModels) <= 0 {
					conf.GoogleAIModels = []string{defaultGoogleAIModel}
				}
				if conf.DesiredLanguage == nil {
					conf.DesiredLanguage = new(defaultDesiredLanguage)
				}
				if conf.FetchFeedsIntervalSeconds <= 0 {
					conf.FetchFeedsIntervalSeconds = defaultFetchFeedsIntervalSeconds
				}
				if conf.FetchFeedsTimeoutSeconds <= 0 {
					conf.FetchFeedsTimeoutSeconds = defaultFetchFeedsTimeoutSeconds
				}
				if conf.ExaoneAccount != nil && conf.ExaoneAccount.KeyFile == "" {
					conf.ExaoneAccount.KeyFile = filepath.Join(conf.DBFilesDirectory, defaultExaoneKeyFilename)
				}

				return conf, nil
			}
		}
	}

	return conf, err
}
