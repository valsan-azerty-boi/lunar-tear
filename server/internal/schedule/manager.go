package schedule

import (
	"fmt"
	"log"
	"sync"
	"time"

	"lunar-tear/server/internal/store"
)

// Manager provides thread-safe access to filtered content catalogs.
// It holds both the full (unfiltered) catalogs and the currently active
// filtered views. On Reload(), it re-reads the config from disk and
// rebuilds the filtered catalogs while holding a write lock.
type Manager struct {
	mu         sync.RWMutex
	configPath string

	// Source of truth
	schedule    *ContentSchedule
	bundleIndex *BundleIndex

	// Full (unfiltered) catalogs — set once at init
	allGachaEntries []store.GachaCatalogEntry

	// Filtered views — rebuilt on each Reload()
	filteredGachaEntries []store.GachaCatalogEntry

	lastUpdatedMillis int64
}

// NewManager creates a schedule manager with the given config path and full catalogs.
// It loads the bundle index from the given JSON file and applies the initial schedule filter.
func NewManager(
	configPath string,
	bundleIndexPath string,
	allGachaEntries []store.GachaCatalogEntry,
) (*Manager, error) {
	sched, err := LoadSchedule(configPath)
	if err != nil {
		return nil, fmt.Errorf("load schedule: %w", err)
	}

	bundleIndex, err := LoadBundleIndex(bundleIndexPath)
	if err != nil {
		return nil, fmt.Errorf("load bundle index: %w", err)
	}

	m := &Manager{
		configPath:        configPath,
		schedule:          sched,
		bundleIndex:       bundleIndex,
		allGachaEntries:   allGachaEntries,
		lastUpdatedMillis: time.Now().UnixMilli(),
	}

	m.rebuildFiltered()

	log.Printf("[schedule] manager initialized: %d/%d gacha entries active, %d bundles enabled",
		len(m.filteredGachaEntries), len(m.allGachaEntries), len(sched.ActiveBundles))

	return m, nil
}

// Reload re-reads the config from disk and rebuilds filtered catalogs.
// This is the hot-reload entry point called by the admin API.
func (m *Manager) Reload() error {
	sched, err := LoadSchedule(m.configPath)
	if err != nil {
		return fmt.Errorf("reload schedule: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.schedule = sched
	m.rebuildFiltered()
	m.lastUpdatedMillis = time.Now().UnixMilli()

	log.Printf("[schedule] reloaded: %d/%d gacha entries active, %d bundles enabled",
		len(m.filteredGachaEntries), len(m.allGachaEntries), len(sched.ActiveBundles))

	return nil
}

// rebuildFiltered rebuilds all filtered catalog views from the full catalogs
// using the current schedule. Must be called with m.mu held for writing.
func (m *Manager) rebuildFiltered() {
	enabledGacha := m.bundleIndex.AllEnabledGachaIds(m.schedule)

	var filtered []store.GachaCatalogEntry
	for _, entry := range m.allGachaEntries {
		if enabledGacha.contains(entry.GachaId) {
			filtered = append(filtered, entry)
		}
	}
	m.filteredGachaEntries = filtered
}

// --- Read accessors (concurrent-safe via RLock) ---

// GachaEntries returns the currently filtered gacha catalog entries.
func (m *Manager) GachaEntries() []store.GachaCatalogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.filteredGachaEntries
}

// Schedule returns a copy of the current content schedule.
func (m *Manager) Schedule() ContentSchedule {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.schedule
}

// BundleIndex returns the bundle index (immutable after init).
func (m *Manager) BundleIndex() *BundleIndex {
	return m.bundleIndex
}

// LastUpdatedMillis returns the timestamp when the schedule was last updated.
func (m *Manager) LastUpdatedMillis() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastUpdatedMillis
}

// Stats returns current statistics for the admin UI.
func (m *Manager) Stats() ScheduleStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return ScheduleStats{
		TotalGachaEntries:    len(m.allGachaEntries),
		ActiveGachaEntries:   len(m.filteredGachaEntries),
		TotalBundles:         len(m.bundleIndex.Bundles),
		ActiveBundles:        len(m.schedule.ActiveBundles),
		UnreleasedEnabled:    m.schedule.UnreleasedEnabled,
		PermanentEventCount:  len(m.bundleIndex.Permanent.EventChapters),
		UnreleasedEventCount: len(m.bundleIndex.Unreleased.EventChapters),
	}
}

// ScheduleStats contains summary statistics for the admin UI.
type ScheduleStats struct {
	TotalGachaEntries    int  `json:"total_gacha_entries"`
	ActiveGachaEntries   int  `json:"active_gacha_entries"`
	TotalBundles         int  `json:"total_bundles"`
	ActiveBundles        int  `json:"active_bundles"`
	UnreleasedEnabled    bool `json:"unreleased_enabled"`
	PermanentEventCount  int  `json:"permanent_event_count"`
	UnreleasedEventCount int  `json:"unreleased_event_count"`
}
