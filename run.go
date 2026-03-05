package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/mmcdole/gofeed"

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

const (
	ignoreItemsPublishedBeforeDays uint = 30 // 1 month

	httpReadTimeout  = 10 * time.Second
	httpWriteTimeout = 30 * time.Second
	httpIdleTimeout  = 60 * time.Second
)

// paywalled sites' urls
var _paywalledSitesURLs = []string{
	"https://www.nytimes.com/",
	"https://www.wsj.com/",
	"https://www.washingtonpost.com/",
	"https://www.economist.com/",
	"https://www.ft.com/",
	"https://www.theguardian.com/",
}

// run with config
func run(conf config) {
	if conf.Verbose {
		log.Printf("> running with config: %s", rf.Prettify(conf))
	}

	// api keys
	apiKeys := []string{}
	if conf.GoogleAIAPIKey != nil {
		apiKeys = append(apiKeys, *conf.GoogleAIAPIKey)
	}
	if len(conf.GoogleAIAPIKeys) > 0 {
		apiKeys = append(apiKeys, conf.GoogleAIAPIKeys...)
	}
	slices.Sort(apiKeys)
	apiKeys = slices.Compact(apiKeys)

	// feed configs for serving
	feedConfs := map[*rf.Client]configRSSFeed{}

	// context for controlling goroutines
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, feedConfig := range conf.RSSFeeds {
		if client, err := rf.NewClientWithDB(
			apiKeys,
			feedConfig.FeedURLs,
			filepath.Join(conf.DBFilesDirectory, feedConfig.CacheFilename),
		); err == nil {
			if len(conf.GoogleAIModels) > 0 {
				client.SetGoogleAIModels(conf.GoogleAIModels)
			}
			client.SetDesiredLanguage(*conf.DesiredLanguage)
			client.SetVerbose(conf.Verbose)

			if conf.Verbose {
				log.Printf(
					"> periodically(interval: %ds) processing feeds from urls: %s",
					conf.FetchFeedsIntervalSeconds,
					strings.Join(feedConfig.FeedURLs, ", "),
				)
			}

			// run periodically (fetch immediately on start, then on interval):
			go func(client *rf.Client) {
				processFeedTick(ctx, client, conf)

				ticker := time.NewTicker(time.Duration(conf.FetchFeedsIntervalSeconds) * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						processFeedTick(ctx, client, conf)
					}
				}
			}(client)

			feedConfs[client] = feedConfig
		} else {
			log.Printf("# failed to create a client: %s", err)
		}
	}

	if len(feedConfs) == 0 {
		log.Printf("# no feed clients were created, exiting")
		return
	}

	// serve RSS feeds
	if conf.Verbose {
		log.Printf("> serving with config: %s", rf.Prettify(conf))
	}
	serve(conf, feedConfs, cancel)
}

// processFeedTick handles a single tick of the feed processing loop
func processFeedTick(parent context.Context, client *rf.Client, conf config) {
	// delete old caches
	client.DeleteOldCachedItems()

	// context with timeout (fetch)
	ctx, cancel := context.WithTimeout(parent, time.Duration(conf.FetchFeedsTimeoutSeconds)*time.Second)
	defer cancel()

	// fetch feeds,
	feeds, err := client.FetchFeeds(
		ctx,
		true,
		ignoreItemsPublishedBeforeDays,
	)
	if err != nil {
		log.Printf("# failed to fetch feeds: %s", err)
		return
	}

	// summarize and cache them,
	if numItems(feeds) > 0 {
		// try creating a new scrapper,
		scrapper := newScrapper()
		if scrapper != nil {
			// scrap + summarize, and cache feeds
			if err := client.SummarizeAndCacheFeeds(feeds, scrapper); err != nil {
				log.Printf("# summary with scrapper failed: %s", err)
			}

			// close the scrapper,
			if err := scrapper.Close(); err != nil {
				log.Printf("# failed to close scrapper: %s", err)
			}
		} else {
			// or just fetch + summarize, and cache feeds
			if err := client.SummarizeAndCacheFeeds(feeds); err != nil {
				log.Printf("# summary failed: %s", err)
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
}

// serve RSS xml
func serve(conf config, feedConfs map[*rf.Client]configRSSFeed, cancelFunc context.CancelFunc) {
	mux := http.NewServeMux()

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

		mux.HandleFunc(path.Join("/", feedConf.ServePath), func(w http.ResponseWriter, r *http.Request) {
			if requestPermitted(r, conf) {
				// fetch cached items,
				items := client.ListCachedItems(true)

				// drop items with failed summaries
				if feedConf.DropItemsWithFailedSummaries {
					items = dropItemsWithFailedSummaries(items)
				}

				// generate xml and serve it
				if bytes, err := client.PublishXML(rssTitle, rssLink, rssDescription, rssAuthor, rssEmail, items); err == nil {
					w.Header().Set("Content-Type", rf.PublishContentType)
					w.Header().Set("Cache-Control", "max-age=60")

					if _, err := w.Write(bytes); err != nil {
						log.Printf("# failed to write data: %s", err)
					}
				} else {
					log.Printf("# failed to serve RSS feeds: %s", err)
					w.WriteHeader(http.StatusInternalServerError)
				}
			} else {
				w.WriteHeader(http.StatusUnauthorized)
			}
		})
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.RSSServerPort),
		Handler:      mux,
		ReadTimeout:  httpReadTimeout,
		WriteTimeout: httpWriteTimeout,
		IdleTimeout:  httpIdleTimeout,
	}

	// graceful shutdown on signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh

		log.Printf("> received signal %s, shutting down...", sig)

		// cancel background goroutines
		cancelFunc()

		// gracefully shutdown the HTTP server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("# failed to shutdown server gracefully: %s", err)
		}
	}()

	// listen and serve
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
func numItems(fs []gofeed.Feed) (num int) {
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
			if strings.HasPrefix(from, "https://www.reddit.com/") {
				// www.reddit.com => old.reddit.com
				return strings.ReplaceAll(from, "www.reddit.com", "old.reddit.com")
			} else if slices.ContainsFunc(_paywalledSitesURLs, func(url string) bool {
				return strings.HasPrefix(from, url)
			}) {
				// use https://www.paywallskip.com/
				return "https://www.paywallskip.com/article?url=" + from
			}

			// default: return it as-is
			return from
		})

		// selector for specific urls
		scrapper.SetSelectorReturner(func(from string) string {
			// x.com => div[data-testid="tweetText"]
			if strings.HasPrefix(from, "https://x.com/") {
				return `article[data-testid='tweet']`
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

// drop items with failed summaries
func dropItemsWithFailedSummaries(items []rf.CachedItem) []rf.CachedItem {
	return slices.DeleteFunc(items, func(item rf.CachedItem) bool {
		return strings.Contains(item.Summary, rf.ErrorPrefixSummaryFailedWithError)
	})
}
