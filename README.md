# summarized-rss-feeds

Fetch RSS feeds, summarize each of them with Google Gemini API, and then serve them as another RSS XML.

## build

```bash
$ go build
```

## configure

Create `config.json` file with content:

```json
{
  "google_ai_api_key": "AIabcdefghijklmnopqrstuvwxyz0123456789",
  "google_ai_model": "gemini-2.0-flash",
  "db_files_dir": "/path/to/summarized-rss-feeds/caches/",
  "desired_language": "Korean",
  "verbose": false,
  //"verbose": true,

  "rss_feeds": [
    {
      "name": "Tech RSS Feeds Summarized",
      "cache_filename": "tech.db",
      "serve_path": "/tech",
      "feed_urls": [
        "https://hnrss.org/newest?points=50",
        "https://lobste.rs/rss",
      ],
      "publish_title": "Summarized RSS Feeds About Technology",
      "publish_link": "https://no-such-domain.com",
      "publish_description": "Tech RSS Feeds Summarized with Google Gemini 1.5 Flash",
      "publish_author": "rss-feeds-summarizer",
      "publish_email": "no-such-email@no-such-domain.com",
    },
  ],
  "fetch_feeds_interval_minutes": 5,
  "permitted_user_agents": [
    "Feedly", // feedly crawler bot's user agent
  ],

  "rss_server_port": 8080,
}
```

This application will poll new feeds from `rss_feeds[].feed_urls` with interval of `fetch_feeds_interval_minutes`.

Then the contents of the new feeds will be fetched using [playwright-go and/or goquery](https://github.com/meinside/simple-scrapper-go).

Fetched contents will be summarized in `desired_language` with your `google_ai_api_key`, and cached in `rss_feeds[].cache_filename` in `db_files_dir`.

Resulting RSS feeds' RSS XML will be served on: yourserver:`rss_server_port`/`rss_feeds[].serve_path`(eg. `localhost:8080/tech`).

Make the URL public and register it on your desired RSS reader, then you'll see your summarized RSS feeds in a few hours.

## run

```bash
$ summarized-rss-feeds /path/to/config.json
```

### run as a service (systemd)

```
[Unit]
Description=Summarized RSS Feeds (Server)
After=syslog.target
After=network.target

[Service]
Type=simple
User=ubuntu
Group=ubuntu
WorkingDirectory=/dir/to/summarized-rss-feeds
ExecStart=/dir/to/summarized-rss-feeds/summarized-rss-feeds /path/to/config.json
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## license

MIT

