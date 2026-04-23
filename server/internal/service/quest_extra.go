package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
)

func (s *QuestServiceServer) StartExtraQuest(ctx context.Context, req *pb.StartExtraQuestRequest) (*pb.StartExtraQuestResponse, error) {
	log.Printf("[QuestService] StartExtraQuest: questId=%d deckNumber=%d", req.QuestId, req.UserDeckNumber)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleExtraQuestStart(user, req.QuestId, req.UserDeckNumber, nowMillis)
	})

	drops := s.engine.BattleDropRewards(req.QuestId)
	pbDrops := make([]*pb.BattleDropReward, len(drops))
	for i, d := range drops {
		pbDrops[i] = &pb.BattleDropReward{
			QuestSceneId:         d.QuestSceneId,
			BattleDropCategoryId: d.BattleDropCategoryId,
			BattleDropEffectId:   1,
		}
	}

	return &pb.StartExtraQuestResponse{
		BattleDropReward: pbDrops,
	}, nil
}

func (s *QuestServiceServer) FinishExtraQuest(ctx context.Context, req *pb.FinishExtraQuestRequest) (*pb.FinishExtraQuestResponse, error) {
	log.Printf("[QuestService] FinishExtraQuest: questId=%d isRetired=%v isAnnihilated=%v", req.QuestId, req.IsRetired, req.IsAnnihilated)

	nowMillis := gametime.NowMillis()
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var outcome questflow.FinishOutcome
	s.users.UpdateUser(userId, func(user *store.UserState) {
		outcome = s.engine.HandleExtraQuestFinish(user, req.QuestId, req.IsRetired, req.IsAnnihilated, nowMillis)
	})

	return &pb.FinishExtraQuestResponse{
		DropReward:                      toProtoRewards(outcome.DropRewards),
		FirstClearReward:                toProtoRewards(outcome.FirstClearRewards),
		MissionClearReward:              toProtoRewards(outcome.MissionClearRewards),
		MissionClearCompleteReward:      toProtoRewards(outcome.MissionClearCompleteRewards),
		IsBigWin:                        outcome.IsBigWin,
		BigWinClearedQuestMissionIdList: outcome.BigWinClearedQuestMissionIds,
		UserStatusCampaignReward:        []*pb.QuestReward{},
	}, nil
}

func (s *QuestServiceServer) RestartExtraQuest(ctx context.Context, req *pb.RestartExtraQuestRequest) (*pb.RestartExtraQuestResponse, error) {
	log.Printf("[QuestService] RestartExtraQuest: questId=%d", req.QuestId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	var deckNumber int32
	s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleExtraQuestRestart(user, req.QuestId, gametime.NowMillis())
		deckNumber = user.Quests[req.QuestId].UserDeckNumber
	})

	drops := s.engine.BattleDropRewards(req.QuestId)
	pbDrops := make([]*pb.BattleDropReward, len(drops))
	for i, d := range drops {
		pbDrops[i] = &pb.BattleDropReward{
			QuestSceneId:         d.QuestSceneId,
			BattleDropCategoryId: d.BattleDropCategoryId,
			BattleDropEffectId:   1,
		}
	}

	return &pb.RestartExtraQuestResponse{
		BattleDropReward: pbDrops,
		DeckNumber:       deckNumber,
	}, nil
}

func (s *QuestServiceServer) UpdateExtraQuestSceneProgress(ctx context.Context, req *pb.UpdateExtraQuestSceneProgressRequest) (*pb.UpdateExtraQuestSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateExtraQuestSceneProgress: questSceneId=%d", req.QuestSceneId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleExtraQuestSceneProgress(user, req.QuestSceneId, gametime.NowMillis())
	})

	return &pb.UpdateExtraQuestSceneProgressResponse{}, nil
}
