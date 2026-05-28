package config

import (
	"errors"
	"os"
	"reflect"
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
