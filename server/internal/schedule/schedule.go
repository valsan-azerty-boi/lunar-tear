package schedule

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// ContentSchedule represents the user-editable content configuration.
type ContentSchedule struct {
	Mode              string            `json:"mode"`
	ActiveBundles     []string          `json:"active_bundles"`
	AlwaysEnabled     AlwaysEnabled     `json:"always_enabled"`
	DisabledOverrides DisabledOverrides `json:"disabled_overrides"`
	UnreleasedEnabled bool              `json:"unreleased_enabled"`
}

// AlwaysEnabled lists IDs that remain active regardless of bundle selection.
type AlwaysEnabled struct {
	EventChapters   []int32 `json:"event_chapters"`
	SideStoryQuests []int32 `json:"side_story_quests"`
}

// DisabledOverrides allows granular opt-out within active bundles.
type DisabledOverrides struct {
	EventChapters []int32 `json:"event_chapters"`
	GachaIds      []int32 `json:"gacha_ids"`
	LoginBonusIds []int32 `json:"login_bonus_ids"`
}

// LoadSchedule reads and parses a content schedule from the given path.
// If the file does not exist, a default empty schedule is returned.
func LoadSchedule(path string) (*ContentSchedule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[schedule] %s not found, using default empty schedule", path)
			return &ContentSchedule{
				Mode:          "custom",
				ActiveBundles: []string{},
			}, nil
		}
		return nil, fmt.Errorf("read schedule %s: %w", path, err)
	}
	var sched ContentSchedule
	if err := json.Unmarshal(data, &sched); err != nil {
		return nil, fmt.Errorf("parse schedule %s: %w", path, err)
	}
	return &sched, nil
}

// SaveSchedule writes the content schedule to the given path.
func SaveSchedule(path string, sched *ContentSchedule) error {
	data, err := json.MarshalIndent(sched, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal schedule: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write schedule %s: %w", path, err)
	}
	return nil
}


// int32Set is a convenience helper for quick membership checks.
type int32Set map[int32]struct{}

func newInt32Set(ids []int32) int32Set {
	s := make(int32Set, len(ids))
	for _, id := range ids {
		s[id] = struct{}{}
	}
	return s
}

func (s int32Set) contains(id int32) bool {
	_, ok := s[id]
	return ok
}

// stringSet is a convenience helper for quick membership checks.
type stringSet map[string]struct{}

func newStringSet(values []string) stringSet {
	s := make(stringSet, len(values))
	for _, v := range values {
		s[v] = struct{}{}
	}
	return s
}

func (s stringSet) contains(v string) bool {
	_, ok := s[v]
	return ok
}
