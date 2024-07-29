package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	rf "github.com/meinside/rss-feeds-go"
	ssg "github.com/meinside/simple-scrapper-go"
)

const (
	defaultPublishTitle       = "Published RSS Feeds"
	defaultPublishLink        = "https://github.com/meinside"
	defaultPublishDescription = "Published RSS Feeds, summarized with Google Gemini API"
	defaultPublishAuthor      = "summarized-rss-feeds"
	defaultPublishEmail       = "noreply@no-such-domain.com"
)

// run with config
func run(conf config) {
	if conf.Verbose {
		log.Printf("> running with config: %s", rf.Prettify(conf))
	}

	feedConfs := map[*rf.Client]configRSSFeed{}

	for _, feedConfig := range conf.RSSFeeds {
		if client, err := rf.NewClientWithDB(conf.GoogleAIAPIKey, feedConfig.FeedURLs, filepath.Join(conf.DBFilesDirectory, feedConfig.CacheFilename)); err == nil {
			client.SetGoogleAIModel(*conf.GoogleAIModel)
			client.SetDesiredLanguage(*conf.DesiredLanguage)
			client.SetVerbose(conf.Verbose)

			if conf.Verbose {
				log.Printf("> periodically processing feeds from urls: %s", strings.Join(feedConfig.FeedURLs, ", "))
			}

			// run periodically:
			go func(client *rf.Client) {
				ticker := time.NewTicker(time.Duration(conf.FetchFeedsIntervalMinutes) * time.Minute)
				for range ticker.C {
					// delete old caches
					client.DeleteOldCachedItems()

					// fetch feeds,
					if feeds, err := client.FetchFeeds(true); err == nil {
						// summarize and cache them,
						if numItems(feeds) > 0 {
							// try creating a new scrapper,
							scrapper := newScrapper()
							if scrapper != nil {
								// scrap + summarize, and cache feeds
								if err := client.SummarizeAndCacheFeeds(feeds, scrapper); err != nil {
									log.Printf("# summary with scrapper failed with some errors: %s", err)
								}

								// close the scrapper,
								if err := scrapper.Close(); err != nil {
									log.Printf("# failed to close scrapper: %s", err)
								}
							} else {
								// or just fetch + summarize, and cache feeds
								if err := client.SummarizeAndCacheFeeds(feeds); err != nil {
									log.Printf("# summary failed with some errors: %s", err)
								}
							}
						}

						// fetch cached (summarized) items,
						items := client.ListCachedItems(false)

						if conf.Verbose {
							log.Printf(">>> fetched %d new item(s).", len(items))
						}

						// and mark them as read
						client.MarkCachedItemsAsRead(items)

						if conf.Verbose {
							log.Printf(">>> marked %d item(s) as read.", len(items))
						}
					} else {
						log.Printf("# failed to fetch feeds: %s", err)
					}
				}
			}(client)

			feedConfs[client] = feedConfig
		} else {
			log.Printf("# failed to create a client: %s", err)
		}
	}

	// serve RSS feeds
	if conf.Verbose {
		log.Printf("> serving with config: %s", rf.Prettify(conf))
	}
	serve(conf, feedConfs)
}

// serve RSS xml
func serve(conf config, feedConfs map[*rf.Client]configRSSFeed) {
	// set http handlers
	for client, feedConf := range feedConfs {
		// get values for publish
		rssTitle := defaultPublishTitle
		if feedConf.PublishTitle != nil {
			rssTitle = *feedConf.PublishTitle
		}
		rssLink := defaultPublishLink
		if feedConf.PublishLink != nil {
			rssLink = *feedConf.PublishLink
		}
		rssDescription := defaultPublishDescription
		if feedConf.PublishDescription != nil {
			rssDescription = *feedConf.PublishDescription
		}
		rssAuthor := defaultPublishAuthor
		if feedConf.PublishAuthor != nil {
			rssAuthor = *feedConf.PublishAuthor
		}
		rssEmail := defaultPublishEmail
		if feedConf.PublishEmail != nil {
			rssEmail = *feedConf.PublishEmail
		}

		http.HandleFunc(path.Join("/", feedConf.ServePath), func(w http.ResponseWriter, r *http.Request) {
			if requestPermitted(r, conf) {
				// fetch cached items,
				items := client.ListCachedItems(true)

				// generate xml and serve it
				if bytes, err := client.PublishXML(rssTitle, rssLink, rssDescription, rssAuthor, rssEmail, items); err == nil {
					w.Header().Set("Content-Type", "application/rss+xml")
					w.Header().Set("Cache-Control", "max-age=60")

					if _, err := io.WriteString(w, string(bytes)); err != nil {
						log.Printf("# failed to write data: %s", err)
					}
				} else {
					log.Printf("# failed to serve RSS feeds: %s", err)
				}
			}
		})
	}

	// listen and serve
	err := http.ListenAndServe(fmt.Sprintf(":%d", conf.RSSServerPort), nil)
	if err != nil {
		log.Printf("# failed to start server: %s", err)
	}
}

// check if given http request is permitted
func requestPermitted(r *http.Request, conf config) bool {
	if len(conf.PermittedUserAgents) > 0 {
		userAgent := r.Header.Get("User-Agent")

		if slices.ContainsFunc(conf.PermittedUserAgents, func(s string) bool {
			return strings.Contains(userAgent, s)
		}) {
			return true
		}

		log.Printf("# dropping access from non-permitted user agent: %s", userAgent)

		return false
	}

	return true

}

// count number of all items of given feeds `fs`
func numItems(fs []feeds.RssFeed) (num int) {
	for _, feed := range fs {
		num += len(feed.Items)
	}
	return num
}

// create a new scrapper
func newScrapper() *ssg.Scrapper {
	if scrapper, err := ssg.NewScrapper(); err == nil {
		// replace urls if needed
		scrapper.SetURLReplacer(func(from string) string {
			// www.reddit.com => old.reddit.com
			if strings.HasPrefix(from, "https://www.reddit.com/") {
				return strings.ReplaceAll(from, "www.reddit.com", "old.reddit.com")
			}
			// default: return it as-is
			return from
		})

		// selector for specific urls
		scrapper.SetSelectorReturner(func(from string) string {
			// x.com => div[data-testid="tweetText"]
			if strings.HasPrefix(from, "https://x.com/") {
				return `div[data-testid="tweetText"]`
			}
			// default: `body`
			return `body`
		})

		return scrapper
	} else {
		log.Printf("# failed to create a new scrapper: %s", err)
	}

	return nil
}
