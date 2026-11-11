# summarized-rss-feeds

Fetch RSS/Atom/JSON feeds, summarize each of them with Google Gemini API and/or [EXAONE API](https://docs.exaone.ai), and then serve them as another RSS XML.

## build

```bash
$ go build
```

## configure

Create `config.json` file with content:

```json
{
  "google_ai_api_keys": [
    "AIabcdefghijklmnopqrstuvwxyz0123456789",
  ],
  "google_ai_models": [
    "gemini-3.6-flash",
  ],
  // (optional) EXAONE account for issuing and renewing an api key automatically, and models
  //"exaone_account": {
  //  "email": "your-exaone-id@example.com",
  //  "password": "your-exaone-password",
  //  "key_file": "/path/to/exaone-key.json", // (optional) default: `exaone-key.json` in `db_files_dir`
  //},
  //"exaone_models": [
  //  "k-exaone_v2.5",
  //],
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
        "https://www.hackster.io/news.atom",
      ],
      "publish_title": "Summarized RSS Feeds About Technology",
      "publish_link": "https://no-such-domain.com",
      "publish_description": "Tech RSS Feeds Summarized with Google Gemini API",
      "publish_author": "rss-feeds-summarizer",
      "publish_email": "no-such-email@no-such-domain.com",
    },
  ],
  "fetch_feeds_interval_seconds": 300,
  "fetch_feeds_timeout_seconds": 60,
  "permitted_user_agents": [
    "Feedly", // feedly crawler bot's user agent
  ],

  "rss_server_port": 8080,
}
```

This application will poll new feeds from `rss_feeds[].feed_urls` with interval of `fetch_feeds_interval_seconds`.

Then the contents of the new feeds will be fetched using [playwright-go and/or goquery](https://github.com/meinside/simple-scrapper-go).

Fetched contents will be summarized in `desired_language` with your `google_ai_api_keys` and/or `exaone_account` (round-robined for text contents), and cached in `rss_feeds[].cache_filename` in `db_files_dir`.

At least one of `google_ai_api_key(s)` or `exaone_account` is required. EXAONE API handles text contents only, so with `exaone_account` alone, files (eg. PDF), YouTube videos, and unfetchable urls will be cached with failed summaries (set `rss_feeds[].drop_items_with_failed_summaries` to `true` for omitting them).

With `exaone_account`, an EXAONE api key is issued (and renewed before it expires in 30 days) with [exaone-authorizer](https://github.com/meinside/exaone-authorizer), and kept in `key_file`. Issuing a key replaces the account's previous one, so do not share the account with other machines. The application exits on start when it cannot get a key, eg. with a wrong password.

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

With `exaone_account`, each start may log in once and repeated failed logins lock the account, so limit the restarts (`Restart=always` restarts even after a clean exit) by adding these to `[Unit]`:

```
StartLimitIntervalSec=1h
StartLimitBurst=3
```

## license

MIT

