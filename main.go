package main

import (
	"encoding/json"
	"log"
	"os"

	rf "github.com/meinside/rss-feeds-go"
)

const (
	defaultGoogleAIModel   = "gemini-1.5-flash-latest"
	defaultDesiredLanguage = "English"

	defaultFetchFeedsIntervalSeconds = 300
)

// config struct
type config struct {
	// System
	GoogleAIAPIKey   string  `json:"google_ai_api_key"`
	GoogleAIModel    *string `json:"google_ai_model,omitempty"`
	DBFilesDirectory string  `json:"db_files_dir"`
	DesiredLanguage  *string `json:"desired_language,omitempty"`
	Verbose          bool    `json:"verbose,omitempty"`

	// RSS
	RSSFeeds                  []configRSSFeed `json:"rss_feeds"`
	FetchFeedsIntervalSeconds int             `json:"fetch_feeds_interval_seconds,omitempty"`
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
				if conf.GoogleAIModel == nil {
					conf.GoogleAIModel = ptr(defaultGoogleAIModel)
				}
				if conf.DesiredLanguage == nil {
					conf.DesiredLanguage = ptr(defaultDesiredLanguage)
				}
				if conf.FetchFeedsIntervalSeconds <= 0 {
					conf.FetchFeedsIntervalSeconds = defaultFetchFeedsIntervalSeconds
				}

				return conf, nil
			}
		}
	}

	return conf, err
}

// return the pointer of given value `v`
func ptr[T any](v T) *T {
	val := v
	return &val
}

// print help message
func printHelp(cmd string) {
	log.Printf(`> Usage:

  * fetch, summarize, cache, and serve RSS feed items
    %[1]s [CONFIG_FILEPATH]
`, cmd)
}

func main() {
	cmd := os.Args[0]
	args := os.Args[1:]

	if len(args) > 0 {
		configFilepath := args[0]
		if conf, err := readConfig(configFilepath); err == nil {
			run(conf)

			os.Exit(0)
		} else {
			log.Printf("> failed to read config: %s", err)
		}
	} else {
		printHelp(cmd)
	}

	os.Exit(1)
}
