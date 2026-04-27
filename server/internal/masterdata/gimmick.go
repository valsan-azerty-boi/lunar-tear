package masterdata

import (
	"fmt"
	"log"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

type gimmickScheduleEntry struct {
	ScheduleId      int32
	StartDatetime   int64
	EndDatetime     int64
	FirstSequenceId int32
	RequiredQuestId int32 // 0 = always active
}

type GimmickCatalog struct {
	schedules []gimmickScheduleEntry
}

func LoadGimmickCatalog(resolver *ConditionResolver) (*GimmickCatalog, error) {
	rows, err := utils.ReadTable[EntityMGimmickSequenceSchedule]("m_gimmick_sequence_schedule")
	if err != nil {
		return nil, fmt.Errorf("load gimmick sequence schedule table: %w", err)
	}

	entries := make([]gimmickScheduleEntry, 0, len(rows))
	for _, r := range rows {
		entry := gimmickScheduleEntry{
			ScheduleId:      r.GimmickSequenceScheduleId,
			StartDatetime:   r.StartDatetime,
			EndDatetime:     r.EndDatetime,
			FirstSequenceId: r.FirstGimmickSequenceId,
		}
		if r.ReleaseEvaluateConditionId != 0 {
			if qid, ok := resolver.RequiredQuestId(r.ReleaseEvaluateConditionId); ok {
				entry.RequiredQuestId = qid
			}
		}
		entries = append(entries, entry)
	}

	log.Printf("gimmick catalog loaded: %d schedules", len(entries))
	return &GimmickCatalog{schedules: entries}, nil
}

func (c *GimmickCatalog) ActiveScheduleKeys(user store.UserState, nowMillis int64) []store.GimmickSequenceKey {
	var keys []store.GimmickSequenceKey
	for _, s := range c.schedules {
		if nowMillis < s.StartDatetime || nowMillis > s.EndDatetime {
			continue
		}
		if s.RequiredQuestId != 0 {
			q, ok := user.Quests[s.RequiredQuestId]
			if !ok || q.QuestStateType != model.UserQuestStateTypeCleared {
				continue
			}
		}
		keys = append(keys, store.GimmickSequenceKey{
			GimmickSequenceScheduleId: s.ScheduleId,
			GimmickSequenceId:         s.FirstSequenceId,
		})
	}
	return keys
}
