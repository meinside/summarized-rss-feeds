// config.go

package main

import (
	"encoding/json"
	"os"

	rf "github.com/meinside/rss-feeds-go"
)

const (
	defaultGoogleAIModel   = `gemini-3-flash-preview`
	defaultDesiredLanguage = `English`

	defaultFetchFeedsIntervalSeconds = 60 * 3 // = 3 minutes
	defaultFetchFeedsTimeoutSeconds  = 60 * 1 // = 1 minute
)

// config struct
type config struct {
	// System
	GoogleAIAPIKey   *string  `json:"google_ai_api_key,omitempty"`
	GoogleAIAPIKeys  []string `json:"google_ai_api_keys,omitempty"`
	GoogleAIModels   []string `json:"google_ai_models,omitempty"`
	DBFilesDirectory string   `json:"db_files_dir"`
	DesiredLanguage  *string  `json:"desired_language,omitempty"`
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

// read config from given `filepath`
func readConfig(filepath string) (conf config, err error) {
	var bytes []byte
	if bytes, err = os.ReadFile(filepath); err == nil {
		if bytes, err = rf.StandardizeJSON(bytes); err == nil {
			if err = json.Unmarshal(bytes, &conf); err == nil {
				// set default values
				if len(conf.GoogleAIModels) <= 0 {
					conf.GoogleAIModels = []string{defaultGoogleAIModel}
				}
				if conf.DesiredLanguage == nil {
					conf.DesiredLanguage = ptr(defaultDesiredLanguage)
				}
				if conf.FetchFeedsIntervalSeconds <= 0 {
					conf.FetchFeedsIntervalSeconds = defaultFetchFeedsIntervalSeconds
				}
				if conf.FetchFeedsTimeoutSeconds <= 0 {
					conf.FetchFeedsTimeoutSeconds = defaultFetchFeedsTimeoutSeconds
				}

				return conf, nil
			}
		}
	}

	return conf, err
}
