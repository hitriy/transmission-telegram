package rutracker

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read testdata %s: %v", name, err)
	}
	decoded, err := decodeWindows1251(data)
	if err != nil {
		t.Fatalf("decode testdata %s: %v", name, err)
	}
	return decoded
}

func TestParseSearch(t *testing.T) {
	html := readTestdata(t, "search_results_page.html")
	results := ParseSearch(html)

	if len(results) != 34 {
		t.Fatalf("expected 34 results, got %d", len(results))
	}

	item := results[33]
	if item.ID != "88068" {
		t.Errorf("id: got %q want %q", item.ID, "88068")
	}
	if item.Title != "[DTSCD][DVDA] Metallica - Black Album - 2005" {
		t.Errorf("title: got %q", item.Title)
	}
	if item.Category != "Рок-музыка (многоканальная музыка)" {
		t.Errorf("category: got %q", item.Category)
	}
	if item.Author != "Меля" {
		t.Errorf("author: got %q", item.Author)
	}
	if item.Seeds != 4 {
		t.Errorf("seeds: got %d want 4", item.Seeds)
	}
	if item.Leeches != 0 {
		t.Errorf("leeches: got %d want 0", item.Leeches)
	}
	if item.Size != 682311748 {
		t.Errorf("size: got %d want 682311748", item.Size)
	}
	if item.Downloads != 7721 {
		t.Errorf("downloads: got %d want 7721", item.Downloads)
	}
	wantRegistered := time.Unix(1162503752, 0)
	if !item.Registered.Equal(wantRegistered) {
		t.Errorf("registered: got %v want %v", item.Registered, wantRegistered)
	}

	if results[0].Seeds != 3 {
		t.Errorf("results[0].seeds: got %d want 3", results[0].Seeds)
	}
	if results[1].Leeches != 3 {
		t.Errorf("results[1].leeches: got %d want 3", results[1].Leeches)
	}
	if results[0].State != "проверено" {
		t.Errorf("results[0].state: got %q want %q", results[0].State, "проверено")
	}
}

func TestParseSearchNoResults(t *testing.T) {
	html := readTestdata(t, "no_results_page.html")
	results := ParseSearch(html)
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %d", len(results))
	}
}

func TestParseMagnetLink(t *testing.T) {
	html := readTestdata(t, "thread.html")
	magnet := ParseMagnetLink(html)
	want := "magnet:?xt=urn:btih:4904EC7AB6106C47B317BA10C688941A9F2202BF&tr=http%3A%2F%2Fbt4.t-ru.org%2Fann%3Fmagnet"
	if magnet != want {
		t.Errorf("magnet: got %q want %q", magnet, want)
	}
}

func TestParseDescription(t *testing.T) {
	html := readTestdata(t, "thread.html")
	desc := ParseDescription(html, 400)
	if desc == "" {
		t.Fatal("expected non-empty description")
	}
	if len([]rune(desc)) > 401 {
		t.Errorf("description too long: %d runes", len([]rune(desc)))
	}
}
