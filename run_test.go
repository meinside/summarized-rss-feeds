package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mmcdole/gofeed"

	rf "github.com/meinside/rss-feeds-go"
)

func TestRequestPermitted(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		permitted []string
		want      bool
	}{
		{
			name:      "no restriction",
			userAgent: "anything",
			permitted: nil,
			want:      true,
		},
		{
			name:      "permitted agent",
			userAgent: "Feedly/1.0",
			permitted: []string{"Feedly"},
			want:      true,
		},
		{
			name:      "non-permitted agent",
			userAgent: "curl/7.68",
			permitted: []string{"Feedly"},
			want:      false,
		},
		{
			name:      "partial match",
			userAgent: "Mozilla/5.0 Feedly-like",
			permitted: []string{"Feedly"},
			want:      true,
		},
		{
			name:      "empty user agent",
			userAgent: "",
			permitted: []string{"Feedly"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.Header.Set("User-Agent", tt.userAgent)

			conf := config{PermittedUserAgents: tt.permitted}

			if got := requestPermitted(r, conf); got != tt.want {
				t.Errorf("requestPermitted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNumItems(t *testing.T) {
	tests := []struct {
		name  string
		feeds []gofeed.Feed
		want  int
	}{
		{
			name:  "empty",
			feeds: nil,
			want:  0,
		},
		{
			name: "single feed with items",
			feeds: []gofeed.Feed{
				{Items: []*gofeed.Item{{}, {}, {}}},
			},
			want: 3,
		},
		{
			name: "multiple feeds",
			feeds: []gofeed.Feed{
				{Items: []*gofeed.Item{{}, {}}},
				{Items: []*gofeed.Item{{}}},
				{Items: nil},
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := numItems(tt.feeds); got != tt.want {
				t.Errorf("numItems() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDropItemsWithFailedSummaries(t *testing.T) {
	items := []rf.CachedItem{
		{Title: "good1", Summary: "This is a good summary"},
		{Title: "failed", Summary: rf.ErrorPrefixSummaryFailedWithError + ": some error"},
		{Title: "good2", Summary: "Another good summary"},
	}

	result := dropItemsWithFailedSummaries(items)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	if result[0].Title != "good1" || result[1].Title != "good2" {
		t.Errorf("unexpected items: %v", result)
	}
}

func TestDropItemsWithFailedSummaries_AllGood(t *testing.T) {
	items := []rf.CachedItem{
		{Title: "a", Summary: "ok"},
		{Title: "b", Summary: "fine"},
	}

	result := dropItemsWithFailedSummaries(items)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
}

func TestDropItemsWithFailedSummaries_Empty(t *testing.T) {
	result := dropItemsWithFailedSummaries(nil)

	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}
