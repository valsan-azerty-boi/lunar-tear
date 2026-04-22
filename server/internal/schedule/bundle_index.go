package schedule

import (
	"log"

	"lunar-tear/server/internal/utils"
)

// BundleIndex maps content into monthly bundles, plus permanent and unreleased buckets.
type BundleIndex struct {
	Bundles    map[string]*Bundle `json:"bundles"`
	Permanent  *Bundle            `json:"permanent"`
	Unreleased *Bundle            `json:"unreleased"`
}

// Bundle represents a group of content that was released together.
type Bundle struct {
	Label         string  `json:"label"`
	EventChapters []int32 `json:"event_chapters"`
	GachaIds      []int32 `json:"gacha_ids"`
	LoginBonuses  []int32 `json:"login_bonuses"`
	SideStories   []int32 `json:"side_stories"` // SideStoryQuestLimitContentIds linked via event chapters
}

func newBundle(label string) *Bundle {
	return &Bundle{
		Label:         label,
		EventChapters: []int32{},
		GachaIds:      []int32{},
		LoginBonuses:  []int32{},
		SideStories:   []int32{},
	}
}

// --- Master data row types for bundle building (read from JSON) ---

type eventQuestChapterRow struct {
	EventQuestChapterId int32 `json:"EventQuestChapterId"`
	StartDatetime       int64 `json:"StartDatetime"`
	EndDatetime         int64 `json:"EndDatetime"`
}

type momBannerRow struct {
	MomBannerId          int32  `json:"MomBannerId"`
	DestinationDomainType int32  `json:"DestinationDomainType"`
	DestinationDomainId  int32  `json:"DestinationDomainId"`
	BannerAssetName      string `json:"BannerAssetName"`
	StartDatetime        int64  `json:"StartDatetime"`
	EndDatetime          int64  `json:"EndDatetime"`
}

type loginBonusRow struct {
	LoginBonusId  int32 `json:"LoginBonusId"`
	StartDatetime int64 `json:"StartDatetime"`
	EndDatetime   int64 `json:"EndDatetime"`
}

type sideStoryLimitContentRow struct {
	SideStoryQuestLimitContentId int32 `json:"SideStoryQuestLimitContentId"`
	CharacterId                  int32 `json:"CharacterId"`
	EventQuestChapterId          int32 `json:"EventQuestChapterId"`
	DifficultyType               int32 `json:"DifficultyType"`
}

// momBannerDomainGacha matches model.MomBannerDomainGacha from the server.
const momBannerDomainGacha int32 = 1

// BuildBundleIndex constructs the bundle index from master data JSON files.
// The JSON files are read from assets/master_data/ (the same location as other master data).
func BuildBundleIndex() *BundleIndex {
	idx := &BundleIndex{
		Bundles:    make(map[string]*Bundle),
		Permanent:  newBundle("Permanent Content"),
		Unreleased: newBundle("Unreleased Content"),
	}

	// --- Event Quest Chapters ---
	eventChapters, err := utils.ReadJSON[eventQuestChapterRow]("EntityMEventQuestChapterTable.json")
	if err != nil {
		log.Printf("[schedule] warning: could not load event quest chapters: %v", err)
	}

	// Build chapter → month lookup for side story mapping
	chapterMonth := make(map[int32]string, len(eventChapters))

	for _, ch := range eventChapters {
		month := ContentMonth(ch.StartDatetime)
		chapterMonth[ch.EventQuestChapterId] = month

		if IsUnreleasedContent(ch.StartDatetime) {
			idx.Unreleased.EventChapters = append(idx.Unreleased.EventChapters, ch.EventQuestChapterId)
			continue
		}
		if IsPermanentContent(ch.EndDatetime) && !IsUnreleasedContent(ch.StartDatetime) {
			// Content with a real start date but permanent end date (e.g. EX chapters, main story events)
			idx.Permanent.EventChapters = append(idx.Permanent.EventChapters, ch.EventQuestChapterId)
			// Also add to the monthly bundle so it can be toggled
		}
		b := idx.getOrCreateBundle(month)
		b.EventChapters = append(b.EventChapters, ch.EventQuestChapterId)
	}

	// --- Gacha Banners (MomBanner type 1 = gacha) ---
	banners, err := utils.ReadJSON[momBannerRow]("EntityMMomBannerTable.json")
	if err != nil {
		log.Printf("[schedule] warning: could not load mom banners: %v", err)
	}

	// Track unique gacha IDs per month (banners can have duplicates)
	gachaSeenPerMonth := make(map[string]int32Set)

	for _, ban := range banners {
		if ban.DestinationDomainType != momBannerDomainGacha {
			continue
		}
		month := ContentMonth(ban.StartDatetime)

		if IsUnreleasedContent(ban.StartDatetime) {
			idx.Unreleased.GachaIds = append(idx.Unreleased.GachaIds, ban.DestinationDomainId)
			continue
		}

		b := idx.getOrCreateBundle(month)
		if gachaSeenPerMonth[month] == nil {
			gachaSeenPerMonth[month] = make(int32Set)
		}
		if !gachaSeenPerMonth[month].contains(ban.DestinationDomainId) {
			b.GachaIds = append(b.GachaIds, ban.DestinationDomainId)
			gachaSeenPerMonth[month][ban.DestinationDomainId] = struct{}{}
		}
	}

	// --- Login Bonuses ---
	loginBonuses, err := utils.ReadJSON[loginBonusRow]("EntityMLoginBonusTable.json")
	if err != nil {
		log.Printf("[schedule] warning: could not load login bonuses: %v", err)
	}

	for _, lb := range loginBonuses {
		month := ContentMonth(lb.StartDatetime)

		if IsUnreleasedContent(lb.StartDatetime) {
			idx.Unreleased.LoginBonuses = append(idx.Unreleased.LoginBonuses, lb.LoginBonusId)
			continue
		}

		b := idx.getOrCreateBundle(month)
		b.LoginBonuses = append(b.LoginBonuses, lb.LoginBonusId)
	}

	// --- Side Stories (linked via event chapters) ---
	ssLimitContent, err := utils.ReadJSON[sideStoryLimitContentRow]("EntityMSideStoryQuestLimitContentTable.json")
	if err != nil {
		log.Printf("[schedule] warning: could not load side story limit content: %v", err)
	}

	for _, ss := range ssLimitContent {
		month, ok := chapterMonth[ss.EventQuestChapterId]
		if !ok {
			continue
		}

		if IsUnreleasedContent(0) {
			// Side stories themselves don't have dates; they inherit from their event chapter.
			// If the chapter is unreleased, so is the side story.
			if month == ContentMonth(0) {
				continue
			}
		}

		b := idx.getOrCreateBundle(month)
		b.SideStories = append(b.SideStories, ss.SideStoryQuestLimitContentId)
	}

	log.Printf("[schedule] bundle index built: %d monthly bundles, %d permanent items, %d unreleased items",
		len(idx.Bundles),
		len(idx.Permanent.EventChapters),
		len(idx.Unreleased.EventChapters)+len(idx.Unreleased.GachaIds)+len(idx.Unreleased.LoginBonuses))

	return idx
}

func (idx *BundleIndex) getOrCreateBundle(month string) *Bundle {
	b, ok := idx.Bundles[month]
	if !ok {
		b = newBundle(month)
		idx.Bundles[month] = b
	}
	return b
}

// AllEnabledEventChapters returns the set of event chapter IDs that should be active
// given the provided schedule.
func (idx *BundleIndex) AllEnabledEventChapters(sched *ContentSchedule) int32Set {
	enabled := make(int32Set)

	// Always-enabled chapters
	for _, id := range sched.AlwaysEnabled.EventChapters {
		enabled[id] = struct{}{}
	}

	// Permanent content is always included
	for _, id := range idx.Permanent.EventChapters {
		enabled[id] = struct{}{}
	}

	// Active bundles
	activeBundles := newStringSet(sched.ActiveBundles)
	for month, bundle := range idx.Bundles {
		if activeBundles.contains(month) {
			for _, id := range bundle.EventChapters {
				enabled[id] = struct{}{}
			}
		}
	}

	// Unreleased
	if sched.UnreleasedEnabled {
		for _, id := range idx.Unreleased.EventChapters {
			enabled[id] = struct{}{}
		}
	}

	// Remove disabled overrides
	disabledChapters := newInt32Set(sched.DisabledOverrides.EventChapters)
	for id := range disabledChapters {
		delete(enabled, id)
	}

	return enabled
}

// AllEnabledGachaIds returns the set of gacha IDs that should be active.
func (idx *BundleIndex) AllEnabledGachaIds(sched *ContentSchedule) int32Set {
	enabled := make(int32Set)

	activeBundles := newStringSet(sched.ActiveBundles)
	for month, bundle := range idx.Bundles {
		if activeBundles.contains(month) {
			for _, id := range bundle.GachaIds {
				enabled[id] = struct{}{}
			}
		}
	}

	if sched.UnreleasedEnabled {
		for _, id := range idx.Unreleased.GachaIds {
			enabled[id] = struct{}{}
		}
	}

	disabledGacha := newInt32Set(sched.DisabledOverrides.GachaIds)
	for id := range disabledGacha {
		delete(enabled, id)
	}

	return enabled
}

// AllEnabledLoginBonuses returns the set of login bonus IDs that should be active.
func (idx *BundleIndex) AllEnabledLoginBonuses(sched *ContentSchedule) int32Set {
	enabled := make(int32Set)

	activeBundles := newStringSet(sched.ActiveBundles)
	for month, bundle := range idx.Bundles {
		if activeBundles.contains(month) {
			for _, id := range bundle.LoginBonuses {
				enabled[id] = struct{}{}
			}
		}
	}

	if sched.UnreleasedEnabled {
		for _, id := range idx.Unreleased.LoginBonuses {
			enabled[id] = struct{}{}
		}
	}

	disabledLB := newInt32Set(sched.DisabledOverrides.LoginBonusIds)
	for id := range disabledLB {
		delete(enabled, id)
	}

	return enabled
}
