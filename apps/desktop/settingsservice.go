package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

// AppSettings are the desktop's own settings, in settings.toml in the
// config folder. They never leave the machine.
type AppSettings struct {
	// QuietColony turns every flavor element off: mood, flavor lines,
	// event letters, sounds and idle animations. Portraits, alerts and
	// letters that ask something stay.
	QuietColony bool `toml:"quiet_colony" json:"quiet_colony"`
	// Sound plays short cues (a task done, a letter, a run's end); off by
	// default.
	Sound bool `toml:"sound" json:"sound"`
	// Volume is 0 to 1.
	Volume float64 `toml:"volume" json:"volume"`
	// SupervisorModel is the model the supervisor chat answers with.
	SupervisorModel string `toml:"supervisor_model" json:"supervisor_model"`
}

// DefaultAppSettings: flavor on, sound off.
func DefaultAppSettings() AppSettings { return AppSettings{Volume: 0.6, SupervisorModel: "sonnet"} }

// SettingsService reads and writes the desktop's settings.
type SettingsService struct {
	path string
	mu   sync.Mutex
}

func NewSettingsService() *SettingsService {
	return &SettingsService{path: filepath.Join(configBase(), "settings.toml")}
}

// Get returns the settings, or the defaults when none are saved.
func (s *SettingsService) Get() (AppSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := DefaultAppSettings()
	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	if err := toml.Unmarshal(raw, &st); err != nil {
		return DefaultAppSettings(), fmt.Errorf("%s: %w", s.path, err)
	}
	if st.SupervisorModel == "" {
		st.SupervisorModel = DefaultAppSettings().SupervisorModel
	}
	return st, nil
}

// Save writes the settings.
func (s *SettingsService) Save(st AppSettings) error {
	if st.Volume < 0 || st.Volume > 1 {
		return errors.New("volume is between 0 and 1")
	}
	st.SupervisorModel = strings.TrimSpace(st.SupervisorModel)
	if st.SupervisorModel == "" {
		st.SupervisorModel = DefaultAppSettings().SupervisorModel
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var buf bytes.Buffer
	buf.WriteString("# The Bakery desktop's own settings. They never leave this machine.\n")
	if err := toml.NewEncoder(&buf).Encode(st); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
