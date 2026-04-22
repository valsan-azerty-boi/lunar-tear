# Content Schedule Manager for Lunar Tear

## Goal

Add a server-side **Content Schedule Manager** that lets users selectively enable/disable time-gated content (events, gacha banners, login bonuses, side stories, etc.) via a web UI served from the existing HTTP port. This replaces the current "everything active at once" approach and lets players progress at their own pace.

## Background

The master data contains **274 event quest chapters**, **307 gacha banners**, **144 side stories**, **116 login bonuses**, and **549 time-limited shops** — spanning the game's full 2019–2024 lifecycle. Currently `patch_masterdata.py` extends all `EndDatetime` fields to 2030, making everything active simultaneously, which overwhelms the client UI.

The data naturally groups into monthly **content bundles** where events, banners, login bonuses, and side stories were released together.

### Side Story ↔ Event Chapter Mapping (Discovered)

Side stories link to event chapters via `EntityMSideStoryQuestLimitContentTable`:

| Month | Event Chapters | Character | Side Stories |
|-------|---------------|-----------|-------------|
| 2022-08 | 500001–500004 | 1019 | 12 (4ch × 3 diff) |
| 2022-10 | 500011–500014 | 1008 | 12 |
| 2022-12 | 500021–500024 | 1007 | 12 |
| 2023-03 | 500031–500034 | 1009 | 12 |
| 2023-04 | 500041–500044 | 1004 | 12 |
| 2023-06 | 500051–500054 | 1013 | 12 |
| 2023-08 | 500061–500064 | 1010 | 12 |
| 2023-10 | 500071–500074 | 1011 | 12 |
| 2023-11 | 500081–500084 | 1015 | 12 |
| 2023-12 | 500091–500094 | 1014 | 12 |
| 2024-01 | 500101–500104 | 1012 | 12 |
| 2024-02 | 500111–500114 | 1006 | 12 |

This means enabling/disabling a bundle's event chapters (500xxx) automatically controls its side stories too — no separate configuration needed.

## Key Design Decisions

1. **Hot-reload from the start** — Saving a new schedule rebuilds catalogs on the fly using `sync.RWMutex` protection. No server restart required.
2. **Unreleased content** — 15 event chapters and 32 login bonuses dated 2099-01 are shown in a dedicated "Unreleased Content" section in the UI, toggleable separately.
3. **Side stories tied to bundles** — Side stories are included in their parent event chapter's bundle automatically via the `SideStoryQuestLimitContent` mapping. When a bundle is disabled, its side stories disappear too.

---

## Proposed Changes

### Phase 1: Content Schedule Config + Hot-Reload Engine

#### [NEW] [content_schedule.json](file:///e:/Emulation/NierReincarnation/lunar-tear/server/content_schedule.json)

Default config file:

```json
{
  "mode": "bundles",
  "active_bundles": ["2022-08", "2022-09", "2022-10"],
  "always_enabled": {
    "event_chapters": [1, 2, 3, 4, 5, 6, 7, 8],
    "side_story_quests": []
  },
  "disabled_overrides": {
    "event_chapters": [],
    "gacha_ids": [],
    "login_bonus_ids": []
  },
  "unreleased_enabled": false
}
```

- `active_bundles`: list of "YYYY-MM" strings; content whose `StartDatetime` falls in these months is enabled
- `always_enabled`: IDs that are always active regardless of bundles (permanent content like EX quests, main story chapters)
- `disabled_overrides`: granular opt-out within active bundles
- `unreleased_enabled`: toggle for 2099-01 dated content

---

#### [NEW] [schedule/schedule.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/internal/schedule/schedule.go)

New `schedule` package:

```go
type ContentSchedule struct {
    Mode              string              `json:"mode"`
    ActiveBundles     []string            `json:"active_bundles"`
    AlwaysEnabled     AlwaysEnabledConfig `json:"always_enabled"`
    DisabledOverrides OverrideConfig      `json:"disabled_overrides"`
    UnreleasedEnabled bool                `json:"unreleased_enabled"`
}

type AlwaysEnabledConfig struct {
    EventChapters   []int32 `json:"event_chapters"`
    SideStoryQuests []int32 `json:"side_story_quests"`
}

type OverrideConfig struct {
    EventChapters []int32 `json:"event_chapters"`
    GachaIds      []int32 `json:"gacha_ids"`
    LoginBonusIds []int32 `json:"login_bonus_ids"`
}
```

Functions:
- `LoadSchedule(path string) (*ContentSchedule, error)` — parse and validate config
- `SaveSchedule(path string, schedule *ContentSchedule) error` — write config
- `ContentMonth(startDatetimeMs int64) string` — convert timestamp to "YYYY-MM" bucket
- `IsUnreleasedContent(startDatetimeMs int64) bool` — check if timestamp is in 2099+ range

---

#### [NEW] [schedule/manager.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/internal/schedule/manager.go)

Hot-reload manager using `sync.RWMutex`:

```go
type ScheduleManager struct {
    mu           sync.RWMutex
    schedule     *ContentSchedule
    configPath   string
    bundleIndex  *BundleIndex

    // Filtered catalogs (rebuilt on schedule change)
    gachaEntries []store.GachaCatalogEntry
    // ... other filtered catalogs
}

func (m *ScheduleManager) Reload() error {
    m.mu.Lock()
    defer m.mu.Unlock()
    // 1. Re-read config from disk
    // 2. Rebuild filtered catalogs from full catalogs
    // 3. Swap pointers atomically
}

func (m *ScheduleManager) GachaEntries() []store.GachaCatalogEntry {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.gachaEntries
}
```

Services that currently receive catalogs directly will instead receive a pointer to the `ScheduleManager` and call getter methods. The `RWMutex` ensures reads are concurrent but writes (reloads) are exclusive.

---

#### [NEW] [schedule/bundle_index.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/internal/schedule/bundle_index.go)

Bundle index built at startup from master data JSON files:

```go
type BundleIndex struct {
    Bundles   map[string]*Bundle   `json:"bundles"`    // keyed by "YYYY-MM"
    Permanent *Bundle              `json:"permanent"`  // always-active content
    Unreleased *Bundle             `json:"unreleased"` // 2099-dated content
}

type Bundle struct {
    Label         string  `json:"label"`
    EventChapters []int32 `json:"event_chapters"`
    GachaIds      []int32 `json:"gacha_ids"`
    LoginBonuses  []int32 `json:"login_bonuses"`
    SideStories   []int32 `json:"side_stories"`  // derived from event chapter mapping
}
```

Built by scanning all master data tables, grouping content by `ContentMonth(StartDatetime)`, and using the `SideStoryQuestLimitContent` table to attach side stories to their parent event chapter bundles.

---

### Phase 2: Server-Side Filtering Integration

#### [MODIFY] [main.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/cmd/lunar-tear/main.go)

- Add `--content-schedule` CLI flag (default: `content_schedule.json`)
- After loading all master data catalogs, create `ScheduleManager` with the full catalogs
- Pass `ScheduleManager` to services instead of raw catalogs where filtering applies

#### [MODIFY] [grpc.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/cmd/lunar-tear/grpc.go)

- `startGRPC` and `registerServices` receive `*schedule.ScheduleManager`
- Services that need filtered data use `manager.GachaEntries()` etc.

#### [MODIFY] [banner.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/internal/service/banner.go)

- `BannerServiceServer` stores `*schedule.ScheduleManager` instead of `[]store.GachaCatalogEntry`
- `GetMamaBanner` calls `manager.GachaEntries()` to get the filtered list

#### [MODIFY] [gacha.go (service)](file:///e:/Emulation/NierReincarnation/lunar-tear/server/internal/service/gacha.go)

- `GachaServiceServer` stores `*schedule.ScheduleManager`
- `GetGachaList`, `GetGacha`, `Draw` use `manager.GachaEntries()`

#### [MODIFY] [loginbonus.go (service)](file:///e:/Emulation/NierReincarnation/lunar-tear/server/internal/service/loginbonus.go)

- Uses `manager.LoginBonusCatalog()` instead of direct catalog reference

#### Filter functions to add in the `schedule` package:

```go
func (m *ScheduleManager) FilterGachaEntries(all []store.GachaCatalogEntry) []store.GachaCatalogEntry
func (m *ScheduleManager) FilterEventChapters(all []masterdata.EventChapterRow) []masterdata.EventChapterRow
func (m *ScheduleManager) FilterLoginBonuses(all *masterdata.LoginBonusCatalog) *masterdata.LoginBonusCatalog
```

---

### Phase 3: Admin Web UI

#### [NEW] [admin.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/internal/service/admin.go)

HTTP handlers under `/admin/`:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/admin/` | GET | Serve the single-page admin UI |
| `/admin/api/status` | GET | Current schedule + bundle index + stats |
| `/admin/api/schedule` | GET | Current schedule config |
| `/admin/api/schedule` | POST | Update schedule config + trigger hot-reload |
| `/admin/api/bundles` | GET | Full bundle index with content counts |
| `/admin/api/reload` | POST | Force reload from disk |

The `POST /admin/api/schedule` endpoint:
1. Validates the new config
2. Writes to `content_schedule.json`
3. Calls `manager.Reload()` to rebuild filtered catalogs
4. Returns success + new stats (active banner count, event count, etc.)

#### [NEW] [admin_ui.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/internal/service/admin_ui.go)

Embedded HTML/JS/CSS using Go's `//go:embed` directive. The UI features:

- **Bundle Timeline**: Horizontal timeline showing monthly content bundles from Aug 2021 → Mar 2024
- **Bundle Cards**: Each month displays its event chapters, gacha banners, login bonuses, and side stories with toggle switches
- **Category Tabs**: Filter view by Events / Gacha / Login Bonuses / Side Stories
- **Unreleased Section**: Dedicated section for 2099-dated content, collapsed by default, with its own toggle
- **Permanent Content**: Always-on section showing permanent quests, displayed as info (non-toggleable)
- **Live Reload**: "Apply Changes" button that POSTs to `/admin/api/schedule` and shows real-time status — **no restart required**
- **Presets**: Quick-select buttons:
  - "Main Story Only" — Disables all events/banners
  - "All Content" — Everything enabled
  - "Chronological" — Enables up to a selected month progressively
- **Stats Bar**: Shows active banner count, event count, etc. Updates live on toggle

#### [MODIFY] [http.go](file:///e:/Emulation/NierReincarnation/lunar-tear/server/cmd/lunar-tear/http.go)

Register `/admin/` route group on the existing HTTP mux.

---

### Phase 4: Bundle Index Generator (Python utility)

#### [NEW] [generate_bundle_index.py](file:///e:/Emulation/NierReincarnation/lunar-scripts/generate_bundle_index.py)

Python script that reads dumped master data and generates `bundle_index.json`:

- Reads `EntityMEventQuestChapterTable.json`, `EntityMMomBannerTable.json`, `EntityMLoginBonusTable.json`, `EntityMSideStoryQuestLimitContentTable.json`
- Groups content by `StartDatetime` month
- Links side stories to event chapters via the limit content table
- Tags permanent content (HUGE EndDatetime sentinels)
- Tags unreleased content (2099 StartDatetime)
- Output used by the Go server and/or the admin UI

> [!TIP]
> This script is a convenience tool. The Go server can also build the bundle index itself from the master data JSON files at startup (and does so for the hot-reload path). The Python script exists for inspection/debugging.

---

## File Summary

| File | Action | Purpose |
|------|--------|---------|
| `server/content_schedule.json` | NEW | User-editable content configuration |
| `server/internal/schedule/schedule.go` | NEW | Config loading/saving, content filtering logic |
| `server/internal/schedule/manager.go` | NEW | Hot-reload manager with RWMutex |
| `server/internal/schedule/bundle_index.go` | NEW | Bundle index builder from master data |
| `server/internal/service/admin.go` | NEW | Admin REST API handlers |
| `server/internal/service/admin_ui.go` | NEW | Embedded web UI (HTML/JS/CSS) |
| `server/cmd/lunar-tear/main.go` | MODIFY | Add --content-schedule flag, wire ScheduleManager |
| `server/cmd/lunar-tear/grpc.go` | MODIFY | Pass ScheduleManager to services |
| `server/cmd/lunar-tear/http.go` | MODIFY | Register /admin/ routes |
| `server/internal/service/banner.go` | MODIFY | Use ScheduleManager for filtered catalog |
| `server/internal/service/gacha.go` | MODIFY | Use ScheduleManager for filtered catalog |
| `server/internal/service/loginbonus.go` | MODIFY | Use ScheduleManager for filtered catalog |
| `lunar-scripts/generate_bundle_index.py` | NEW | Python bundle index generator |

---

## Verification Plan

### Automated Tests

```bash
# Phase 1: Verify build
cd server && go build ./...

# Phase 1: Unit tests for schedule package
go test ./internal/schedule/...

# Phase 2: Verify filtered catalogs
go test ./internal/service/... -run TestFilteredGacha

# Phase 3: Verify admin UI serves
curl http://localhost:8080/admin/
curl http://localhost:8080/admin/api/bundles
```

### Manual Verification (Browser)

1. Start the server with default schedule (a few bundles enabled)
2. Open `http://localhost:8080/admin/` — verify the timeline UI loads
3. Toggle a bundle off → click "Apply Changes" → verify success response
4. Open gacha list in-game → verify banner count matches the enabled bundles
5. Enable "Unreleased Content" → verify 2099 items appear
6. Use "Chronological" preset up to 2023-06 → verify only content through that date is active
7. Disable a specific banner within an active bundle → verify it disappears from the gacha list while the rest remains

### Integration Check

1. Verify hot-reload: toggle bundles while a game client is connected; next gacha list request should reflect the change
2. Verify side stories: disable a month with side stories (e.g., 2022-08) → verify the 500001–500004 chapters and their 12 side stories are hidden
3. Verify persistence: restart the server → verify the last saved schedule is still active
