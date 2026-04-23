package userdata

import (
	"sort"

	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

func init() {
	register("IUserGimmick", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedGimmickRecords(user)...)
		return s
	})
	register("IUserGimmickOrnamentProgress", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedGimmickOrnamentProgressRecords(user)...)
		return s
	})
	register("IUserGimmickSequence", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedGimmickSequenceRecords(user)...)
		return s
	})
	register("IUserGimmickUnlock", func(user store.UserState) string {
		s, _ := utils.EncodeJSONMaps(sortedGimmickUnlockRecords(user)...)
		return s
	})
}

func sortedGimmickRecords(user store.UserState) []map[string]any {
	keys := make([]store.GimmickKey, 0, len(user.Gimmick.Progress))
	for key := range user.Gimmick.Progress {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareGimmickKey(keys[i], keys[j]) < 0
	})

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Gimmick.Progress[key]
		records = append(records, map[string]any{
			"userId":                    user.UserId,
			"gimmickSequenceScheduleId": row.Key.GimmickSequenceScheduleId,
			"gimmickSequenceId":         row.Key.GimmickSequenceId,
			"gimmickId":                 row.Key.GimmickId,
			"isGimmickCleared":          row.IsGimmickCleared,
			"startDatetime":             row.StartDatetime,
			"latestVersion":             row.LatestVersion,
		})
	}
	return records
}

func sortedGimmickOrnamentProgressRecords(user store.UserState) []map[string]any {
	keys := make([]store.GimmickOrnamentKey, 0, len(user.Gimmick.OrnamentProgress))
	for key := range user.Gimmick.OrnamentProgress {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareGimmickOrnamentKey(keys[i], keys[j]) < 0
	})

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Gimmick.OrnamentProgress[key]
		records = append(records, map[string]any{
			"userId":                    user.UserId,
			"gimmickSequenceScheduleId": row.Key.GimmickSequenceScheduleId,
			"gimmickSequenceId":         row.Key.GimmickSequenceId,
			"gimmickId":                 row.Key.GimmickId,
			"gimmickOrnamentIndex":      row.Key.GimmickOrnamentIndex,
			"progressValueBit":          row.ProgressValueBit,
			"baseDatetime":              row.BaseDatetime,
			"latestVersion":             row.LatestVersion,
		})
	}
	return records
}

func sortedGimmickSequenceRecords(user store.UserState) []map[string]any {
	keys := make([]store.GimmickSequenceKey, 0, len(user.Gimmick.Sequences))
	for key := range user.Gimmick.Sequences {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].GimmickSequenceScheduleId != keys[j].GimmickSequenceScheduleId {
			return keys[i].GimmickSequenceScheduleId < keys[j].GimmickSequenceScheduleId
		}
		return keys[i].GimmickSequenceId < keys[j].GimmickSequenceId
	})

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Gimmick.Sequences[key]
		records = append(records, map[string]any{
			"userId":                    user.UserId,
			"gimmickSequenceScheduleId": row.Key.GimmickSequenceScheduleId,
			"gimmickSequenceId":         row.Key.GimmickSequenceId,
			"isGimmickSequenceCleared":  row.IsGimmickSequenceCleared,
			"clearDatetime":             row.ClearDatetime,
			"latestVersion":             row.LatestVersion,
		})
	}
	return records
}

func sortedGimmickUnlockRecords(user store.UserState) []map[string]any {
	keys := make([]store.GimmickKey, 0, len(user.Gimmick.Unlocks))
	for key := range user.Gimmick.Unlocks {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return compareGimmickKey(keys[i], keys[j]) < 0
	})

	records := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		row := user.Gimmick.Unlocks[key]
		records = append(records, map[string]any{
			"userId":                    user.UserId,
			"gimmickSequenceScheduleId": row.Key.GimmickSequenceScheduleId,
			"gimmickSequenceId":         row.Key.GimmickSequenceId,
			"gimmickId":                 row.Key.GimmickId,
			"isUnlocked":                row.IsUnlocked,
			"latestVersion":             row.LatestVersion,
		})
	}
	return records
}
