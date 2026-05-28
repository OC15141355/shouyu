package config

import (
	"errors"
	"os"
	"reflect"
	"strconv"
	"testing"
)

func TestLoad_Valid(t *testing.T) {
	cfg, err := Load("testdata/tiles_valid.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Brand.Name != "Fran" {
		t.Fatalf("brand = %q", cfg.Brand.Name)
	}
	if len(cfg.Tiles) != 2 {
		t.Fatalf("tile count = %d", len(cfg.Tiles))
	}
	if cfg.Tiles[0].ID != "fran" || cfg.Tiles[0].Href != "https://fran.yagura.dev" {
		t.Fatalf("tile 0 = %+v", cfg.Tiles[0])
	}
}

func TestLoad_RejectsMissingHref(t *testing.T) {
	if _, err := Load("testdata/tiles_missing_href.yaml"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("testdata/nope.yaml")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want not-exist error, got %v", err)
	}
}

func TestLoad_RejectsNonHTTPScheme(t *testing.T) {
	cases := []struct {
		name string
		href string
	}{
		{"javascript", "javascript:alert(1)"},
		{"data", "data:text/html,<script>"},
		{"file", "file:///etc/passwd"},
		{"scheme-relative", "//evil.example/x"},
		{"bare path", "/relative/path"},
		{"ftp", "ftp://example.com/file"},
		{"empty-after-strip", " "}, // whitespace is non-empty, so current check passes; must still reject
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			yml := "tiles:\n  - {id: x, name: X, href: " + strconv.Quote(c.href) + ", visible_to_groups: [family]}\n"
			path := t.TempDir() + "/tiles.yaml"
			if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err == nil {
				t.Fatalf("expected error for href %q", c.href)
			}
		})
	}
}

func TestLoad_AcceptsHTTPAndHTTPS(t *testing.T) {
	cases := []string{
		"http://example.com",
		"https://example.com/x",
		"https://sub.example.com:8443/path?q=1",
	}
	for _, h := range cases {
		t.Run(h, func(t *testing.T) {
			yml := "tiles:\n  - {id: x, name: X, href: " + strconv.Quote(h) + ", visible_to_groups: [family]}\n"
			path := t.TempDir() + "/tiles.yaml"
			if err := os.WriteFile(path, []byte(yml), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(path); err != nil {
				t.Fatalf("unexpected error for href %q: %v", h, err)
			}
		})
	}
}

func TestFilterByGroups(t *testing.T) {
	cfg := &Config{Tiles: []Tile{
		{ID: "a", VisibleToGroups: []string{"family"}},
		{ID: "b", VisibleToGroups: []string{"admin"}},
		{ID: "c", VisibleToGroups: []string{"family", "admin"}},
	}}
	got := cfg.FilterByGroups([]string{"family"})
	want := []string{"a", "c"}
	gotIDs := make([]string, len(got))
	for i, t := range got {
		gotIDs[i] = t.ID
	}
	if !reflect.DeepEqual(gotIDs, want) {
		t.Fatalf("got %v, want %v", gotIDs, want)
	}
}
