package main

import (
	"path/filepath"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	s := &SettingsService{path: filepath.Join(t.TempDir(), "settings.toml")}
	got, err := s.Get()
	if err != nil || got != DefaultAppSettings() || got.Sound || got.QuietColony {
		t.Fatalf("fresh settings = %+v, %v; want sound off and flavor on", got, err)
	}
	want := AppSettings{QuietColony: true, Sound: true, Volume: 0.3}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(); got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	if err := s.Save(AppSettings{Volume: 2}); err == nil {
		t.Fatal("a volume of 2 was saved")
	}
}
