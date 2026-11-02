package app

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSettingsRoundTripAndCorruptFilePreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "settings.json")
	s := defaultSettings()
	s.FontSize, s.Opacity = 18, 0.4
	s.Profiles = append(s.Profiles, profile{ID: "custom", Name: "Custom", Command: []string{"agent", "--flag", "some value"}})
	if err := saveSettings(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := loadSettings(path)
	if err != nil || !reflect.DeepEqual(got, s) {
		t.Fatalf("roundtrip mismatch: %v", err)
	}
	bad := []byte(`{"background":"not-a-color"}`)
	if err := os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	got, err = loadSettings(path)
	if err == nil || got.Background != defaultSettings().Background {
		t.Fatal("invalid config should fall back safely")
	}
	b, _ := os.ReadFile(path)
	if string(b) != string(bad) {
		t.Fatal("invalid user config must remain available for repair")
	}
}

func TestSettingsRejectDuplicateProfilesBeforeWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := defaultSettings()
	s.Profiles = append(s.Profiles, s.Profiles[0])
	if err := saveSettings(path, s); err == nil {
		t.Fatal("expected duplicate profile error")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("invalid settings must not be written")
	}
}
