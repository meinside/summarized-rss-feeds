package main

import (
	"os"
	"strings"
	"testing"

	rf "github.com/meinside/rss-feeds-go"
)

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()

	f, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(f.Name()) })

	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	return f.Name()
}

const validConfig = `{
	"google_ai_api_keys": ["key1"],
	"db_files_dir": "/tmp",
	"rss_feeds": [
		{
			"name": "test",
			"cache_filename": "test.db",
			"serve_path": "/test",
			"feed_urls": ["https://example.com/rss"]
		}
	],
	"rss_server_port": 9090
}`

func TestReadConfig(t *testing.T) {
	path := writeTestConfig(t, validConfig)

	conf, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error: %s", err)
	}

	if conf.RSSServerPort != 9090 {
		t.Errorf("expected port 9090, got %d", conf.RSSServerPort)
	}
	if conf.DBFilesDirectory != "/tmp" {
		t.Errorf("expected db_files_dir /tmp, got %s", conf.DBFilesDirectory)
	}
	if len(conf.RSSFeeds) != 1 {
		t.Errorf("expected 1 rss feed, got %d", len(conf.RSSFeeds))
	}
}

func TestReadConfig_Defaults(t *testing.T) {
	path := writeTestConfig(t, validConfig)

	conf, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error: %s", err)
	}

	if len(conf.GoogleAIModels) != 1 || conf.GoogleAIModels[0] != defaultGoogleAIModel {
		t.Errorf("expected default model %s, got %v", defaultGoogleAIModel, conf.GoogleAIModels)
	}
	if conf.DesiredLanguage == nil || *conf.DesiredLanguage != defaultDesiredLanguage {
		t.Errorf("expected default language %s, got %v", defaultDesiredLanguage, conf.DesiredLanguage)
	}
	if conf.FetchFeedsIntervalSeconds != defaultFetchFeedsIntervalSeconds {
		t.Errorf("expected interval %d, got %d", defaultFetchFeedsIntervalSeconds, conf.FetchFeedsIntervalSeconds)
	}
	if conf.FetchFeedsTimeoutSeconds != defaultFetchFeedsTimeoutSeconds {
		t.Errorf("expected timeout %d, got %d", defaultFetchFeedsTimeoutSeconds, conf.FetchFeedsTimeoutSeconds)
	}
}

func TestReadConfig_InvalidPath(t *testing.T) {
	_, err := readConfig("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestReadConfig_InvalidJSON(t *testing.T) {
	path := writeTestConfig(t, "not json")

	_, err := readConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestReadConfig_MissingAPIKeys(t *testing.T) {
	content := `{
		"db_files_dir": "/tmp",
		"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}],
		"rss_server_port": 8080
	}`
	path := writeTestConfig(t, content)

	_, err := readConfig(path)
	if err == nil {
		t.Fatal("expected error for missing API keys, got nil")
	}
	if !strings.Contains(err.Error(), "google_ai_api_key") {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestReadConfig_MissingDBDir(t *testing.T) {
	content := `{
		"google_ai_api_keys": ["key1"],
		"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}],
		"rss_server_port": 8080
	}`
	path := writeTestConfig(t, content)

	_, err := readConfig(path)
	if err == nil {
		t.Fatal("expected error for missing db_files_dir, got nil")
	}
	if !strings.Contains(err.Error(), "db_files_dir") {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestReadConfig_EmptyRSSFeeds(t *testing.T) {
	content := `{
		"google_ai_api_keys": ["key1"],
		"db_files_dir": "/tmp",
		"rss_feeds": [],
		"rss_server_port": 8080
	}`
	path := writeTestConfig(t, content)

	_, err := readConfig(path)
	if err == nil {
		t.Fatal("expected error for empty rss_feeds, got nil")
	}
	if !strings.Contains(err.Error(), "rss_feeds") {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestReadConfig_InvalidPort(t *testing.T) {
	content := `{
		"google_ai_api_keys": ["key1"],
		"db_files_dir": "/tmp",
		"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}]
	}`
	path := writeTestConfig(t, content)

	_, err := readConfig(path)
	if err == nil {
		t.Fatal("expected error for missing rss_server_port, got nil")
	}
	if !strings.Contains(err.Error(), "rss_server_port") {
		t.Errorf("unexpected error message: %s", err)
	}
}

func TestReadConfig_ExaoneAPIKeys(t *testing.T) {
	content := `{
		"google_ai_api_keys": ["key1"],
		"exaone_api_keys": ["exaone-key1", "exaone-key2"],
		"exaone_models": ["exaone-model"],
		"db_files_dir": "/tmp",
		"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}],
		"rss_server_port": 8080
	}`
	path := writeTestConfig(t, content)

	conf, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error: %s", err)
	}

	if len(conf.ExaoneAPIKeys) != 2 || conf.ExaoneAPIKeys[0] != "exaone-key1" || conf.ExaoneAPIKeys[1] != "exaone-key2" {
		t.Errorf("expected 2 exaone api keys, got %v", conf.ExaoneAPIKeys)
	}
	if len(conf.ExaoneModels) != 1 || conf.ExaoneModels[0] != "exaone-model" {
		t.Errorf("expected exaone model, got %v", conf.ExaoneModels)
	}
}

func TestReadConfig_ExaoneAPIKeysOnly(t *testing.T) {
	content := `{
		"exaone_api_keys": ["exaone-key1"],
		"db_files_dir": "/tmp",
		"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}],
		"rss_server_port": 8080
	}`
	path := writeTestConfig(t, content)

	conf, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error: %s", err)
	}
	if conf.GoogleAIAPIKey != nil || len(conf.GoogleAIAPIKeys) != 0 {
		t.Errorf("expected no google ai api keys, got %v, %v", conf.GoogleAIAPIKey, conf.GoogleAIAPIKeys)
	}
	if len(conf.ExaoneAPIKeys) != 1 || conf.ExaoneAPIKeys[0] != "exaone-key1" {
		t.Errorf("expected 1 exaone api key, got %v", conf.ExaoneAPIKeys)
	}
}

func TestConfigRedacted(t *testing.T) {
	googleKey := "google-single-key-0123456789"
	conf := config{
		GoogleAIAPIKey:  &googleKey,
		GoogleAIAPIKeys: []string{"google-key-abcdefghijklmn", "short"},
		ExaoneAPIKeys:   []string{"exaone-key-opqrstuvwxyz"},
	}

	logged := rf.Prettify(conf.redacted())

	for _, key := range []string{googleKey, "google-key-abcdefghijklmn", "short", "exaone-key-opqrstuvwxyz"} {
		if strings.Contains(logged, key) {
			t.Errorf("api key '%s' was not masked: %s", key, logged)
		}
	}
	for _, suffix := range []string{"********6789", "********klmn", "********wxyz"} {
		if !strings.Contains(logged, suffix) {
			t.Errorf("expected masked key '%s' in: %s", suffix, logged)
		}
	}

	// the original config must be left untouched
	if *conf.GoogleAIAPIKey != googleKey ||
		conf.GoogleAIAPIKeys[0] != "google-key-abcdefghijklmn" ||
		conf.GoogleAIAPIKeys[1] != "short" ||
		conf.ExaoneAPIKeys[0] != "exaone-key-opqrstuvwxyz" {
		t.Errorf("original config was modified: %+v", conf)
	}
}

func TestReadConfig_SingleAPIKey(t *testing.T) {
	content := `{
		"google_ai_api_key": "single-key",
		"db_files_dir": "/tmp",
		"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}],
		"rss_server_port": 8080
	}`
	path := writeTestConfig(t, content)

	conf, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error: %s", err)
	}

	if conf.GoogleAIAPIKey == nil || *conf.GoogleAIAPIKey != "single-key" {
		t.Errorf("expected single API key, got %v", conf.GoogleAIAPIKey)
	}
}
