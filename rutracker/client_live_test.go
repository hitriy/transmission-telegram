//go:build integration

package rutracker

import (
	"os"
	"testing"
)

func TestLiveLoginAndSearch(t *testing.T) {
	user := os.Getenv("RT_USER")
	pass := os.Getenv("RT_PASS")
	if user == "" || pass == "" {
		t.Skip("RT_USER and RT_PASS required for integration test")
	}

	client := NewClient(user, pass)
	if err := client.Login(); err != nil {
		t.Fatalf("login: %v", err)
	}

	results, err := client.Search("ubuntu")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected search results")
	}
	t.Logf("found %d results, first: %s", len(results), results[0].Title)
}
