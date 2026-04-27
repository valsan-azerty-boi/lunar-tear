package questflow

import (
	"fmt"
	"log"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func (h *QuestHandler) HandleExtraQuestStart(user *store.UserState, questId, userDeckNumber int32, nowMillis int64) {
	quest, ok := h.QuestById[questId]
	if !ok {
		panic(fmt.Sprintf("unknown questId=%d for HandleExtraQuestStart", questId))
	}

	h.initQuestState(user, questId)

	if quest.Stamina > 0 {
		maxMillis := h.MaxStaminaByLevel[user.Status.Level] * 1000
		store.ConsumeStamina(user, quest.Stamina, maxMillis, nowMillis)
	}

	questState := user.Quests[questId]
	questState.UserDeckNumber = userDeckNumber
	questState.QuestStateType = model.UserQuestStateTypeActive
	questState.LatestStartDatetime = nowMillis
	user.Quests[questId] = questState

	user.ExtraQuest.CurrentQuestId = questId
	if sceneIds := h.SceneIdsByQuestId[questId]; len(sceneIds) > 0 {
		user.ExtraQuest.CurrentQuestSceneId = sceneIds[0]
		user.ExtraQuest.HeadQuestSceneId = sceneIds[0]
	}
}

func (h *QuestHandler) HandleExtraQuestFinish(user *store.UserState, questId int32, isRetired, isAnnihilated bool, nowMillis int64) FinishOutcome {
	quest, ok := h.QuestById[questId]
	if !ok {
		panic(fmt.Sprintf("unknown questId=%d for HandleExtraQuestFinish", questId))
	}

	outcome := h.evaluateFinishOutcome(user, questId)
	if !isRetired {
		h.applyQuestVictory(user, questId, &outcome, nowMillis)
	}

	if isRetired && !isAnnihilated && quest.Stamina > 1 {
		refund := quest.Stamina - 1
		maxMillis := h.MaxStaminaByLevel[user.Status.Level] * 1000
		store.RecoverStamina(user, refund*1000, maxMillis, nowMillis)
	}

	user.ExtraQuest.CurrentQuestId = 0
	user.ExtraQuest.CurrentQuestSceneId = 0
	user.ExtraQuest.HeadQuestSceneId = 0

	h.clearQuestMissions(user, questId, nowMillis)

	return outcome
}

func (h *QuestHandler) HandleExtraQuestRestart(user *store.UserState, questId int32, nowMillis int64) {
	h.HandleQuestRestart(user, questId, nowMillis)

	user.ExtraQuest.CurrentQuestId = questId
}

func (h *QuestHandler) HandleExtraQuestSceneProgress(user *store.UserState, questSceneId int32, nowMillis int64) {
	scene, ok := h.SceneById[questSceneId]
	if !ok {
		log.Printf("[HandleExtraQuestSceneProgress] unknown sceneId=%d, skipping", questSceneId)
		return
	}

	user.ExtraQuest.CurrentQuestSceneId = questSceneId
	if h.isSceneAhead(questSceneId, user.ExtraQuest.HeadQuestSceneId) {
		user.ExtraQuest.HeadQuestSceneId = questSceneId
	}

	h.applySceneGrants(user, questSceneId, nowMillis)

	if model.QuestResultType(scene.QuestResultType) == model.QuestResultTypeHalfResult {
		h.clearQuestMissions(user, scene.QuestId, nowMillis)
	}
}
