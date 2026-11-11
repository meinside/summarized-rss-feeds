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

func TestReadConfig_ExaoneModels(t *testing.T) {
	content := `{
		"google_ai_api_keys": ["key1"],
		"exaone_account": {"email": "user@example.com", "password": "the-password"},
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

	if len(conf.ExaoneModels) != 1 || conf.ExaoneModels[0] != "exaone-model" {
		t.Errorf("expected exaone model, got %v", conf.ExaoneModels)
	}
}

func TestReadConfig_ExaoneAccount(t *testing.T) {
	content := `{
		"exaone_account": {"email": "user@example.com", "password": "the-password"},
		"db_files_dir": "/tmp/caches",
		"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}],
		"rss_server_port": 8080
	}`
	path := writeTestConfig(t, content)

	conf, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error: %s", err)
	}
	// without google ai api keys
	if conf.GoogleAIAPIKey != nil || len(conf.GoogleAIAPIKeys) != 0 {
		t.Errorf("expected no google ai api keys, got %v, %v", conf.GoogleAIAPIKey, conf.GoogleAIAPIKeys)
	}
	if conf.ExaoneAccount == nil || conf.ExaoneAccount.Email != "user@example.com" || conf.ExaoneAccount.Password != "the-password" {
		t.Fatalf("unexpected exaone account: %+v", conf.ExaoneAccount)
	}
	if want := "/tmp/caches/" + defaultExaoneKeyFilename; conf.ExaoneAccount.KeyFile != want {
		t.Errorf("expected default key file %s, got %s", want, conf.ExaoneAccount.KeyFile)
	}
}

func TestReadConfig_ExaoneAccountKeyFile(t *testing.T) {
	content := `{
		"exaone_account": {"email": "user@example.com", "password": "the-password", "key_file": "/somewhere/key.json"},
		"db_files_dir": "/tmp/caches",
		"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}],
		"rss_server_port": 8080
	}`
	path := writeTestConfig(t, content)

	conf, err := readConfig(path)
	if err != nil {
		t.Fatalf("readConfig() error: %s", err)
	}
	if conf.ExaoneAccount.KeyFile != "/somewhere/key.json" {
		t.Errorf("expected given key file, got %s", conf.ExaoneAccount.KeyFile)
	}
}

func TestReadConfig_ExaoneAccountInvalid(t *testing.T) {
	for name, tc := range map[string]struct {
		fields  string
		wantErr string
	}{
		"without password": {
			fields:  `"exaone_account": {"email": "user@example.com"},`,
			wantErr: "'email' and 'password'",
		},
		"without email": {
			fields:  `"exaone_account": {"password": "the-password"},`,
			wantErr: "'email' and 'password'",
		},
	} {
		t.Run(name, func(t *testing.T) {
			content := `{
				` + tc.fields + `
				"db_files_dir": "/tmp",
				"rss_feeds": [{"name":"t","cache_filename":"t.db","serve_path":"/t","feed_urls":["https://example.com/rss"]}],
				"rss_server_port": 8080
			}`
			path := writeTestConfig(t, content)

			if _, err := readConfig(path); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestConfigRedacted(t *testing.T) {
	googleKey := "google-single-key-0123456789"
	conf := config{
		GoogleAIAPIKey:  &googleKey,
		GoogleAIAPIKeys: []string{"google-key-abcdefghijklmn", "short"},
		ExaoneAccount:   &configExaoneAccount{Email: "user@example.com", Password: "the-exaone-password"},
	}

	logged := rf.Prettify(conf.redacted())

	for _, key := range []string{googleKey, "google-key-abcdefghijklmn", "short", "the-exaone-password"} {
		if strings.Contains(logged, key) {
			t.Errorf("api key '%s' was not masked: %s", key, logged)
		}
	}
	for _, suffix := range []string{"********6789", "********klmn"} {
		if !strings.Contains(logged, suffix) {
			t.Errorf("expected masked key '%s' in: %s", suffix, logged)
		}
	}

	// the original config must be left untouched
	if *conf.GoogleAIAPIKey != googleKey ||
		conf.GoogleAIAPIKeys[0] != "google-key-abcdefghijklmn" ||
		conf.GoogleAIAPIKeys[1] != "short" ||
		conf.ExaoneAccount.Password != "the-exaone-password" {
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
