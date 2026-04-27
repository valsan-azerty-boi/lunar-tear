package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

func (s *QuestServiceServer) StartEventQuest(ctx context.Context, req *pb.StartEventQuestRequest) (*pb.StartEventQuestResponse, error) {
	log.Printf("[QuestService] StartEventQuest: chapterId=%d questId=%d isBattleOnly=%v", req.EventQuestChapterId, req.QuestId, req.IsBattleOnly)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleEventQuestStart(user, req.EventQuestChapterId, req.QuestId, req.IsBattleOnly, req.UserDeckNumber, nowMillis)
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

	return &pb.StartEventQuestResponse{
		BattleDropReward: pbDrops,
	}, nil
}

func (s *QuestServiceServer) FinishEventQuest(ctx context.Context, req *pb.FinishEventQuestRequest) (*pb.FinishEventQuestResponse, error) {
	log.Printf("[QuestService] FinishEventQuest: chapterId=%d questId=%d isRetired=%v isAnnihilated=%v", req.EventQuestChapterId, req.QuestId, req.IsRetired, req.IsAnnihilated)

	nowMillis := gametime.NowMillis()
	userId := CurrentUserId(ctx, s.users, s.sessions)
	var outcome questflow.FinishOutcome
	s.users.UpdateUser(userId, func(user *store.UserState) {
		outcome = s.engine.HandleEventQuestFinish(user, req.EventQuestChapterId, req.QuestId, req.IsRetired, req.IsAnnihilated, nowMillis)
	})

	return &pb.FinishEventQuestResponse{
		DropReward:                      toProtoRewards(outcome.DropRewards),
		FirstClearReward:                toProtoRewards(outcome.FirstClearRewards),
		MissionClearReward:              toProtoRewards(outcome.MissionClearRewards),
		MissionClearCompleteReward:      toProtoRewards(outcome.MissionClearCompleteRewards),
		AutoOrbitResult:                 []*pb.QuestReward{},
		IsBigWin:                        outcome.IsBigWin,
		BigWinClearedQuestMissionIdList: outcome.BigWinClearedQuestMissionIds,
		UserStatusCampaignReward:        []*pb.QuestReward{},
	}, nil
}

func (s *QuestServiceServer) RestartEventQuest(ctx context.Context, req *pb.RestartEventQuestRequest) (*pb.RestartEventQuestResponse, error) {
	log.Printf("[QuestService] RestartEventQuest: chapterId=%d questId=%d", req.EventQuestChapterId, req.QuestId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleEventQuestRestart(user, req.EventQuestChapterId, req.QuestId, gametime.NowMillis())
	})

	return &pb.RestartEventQuestResponse{
		BattleDropReward: []*pb.BattleDropReward{},
	}, nil
}

func (s *QuestServiceServer) UpdateEventQuestSceneProgress(ctx context.Context, req *pb.UpdateEventQuestSceneProgressRequest) (*pb.UpdateEventQuestSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateEventQuestSceneProgress: questSceneId=%d", req.QuestSceneId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleEventQuestSceneProgress(user, req.QuestSceneId, gametime.NowMillis())
	})

	return &pb.UpdateEventQuestSceneProgressResponse{}, nil
}

const defaultGuerrillaFreeOpenMinutes = int32(60)

func (s *QuestServiceServer) StartGuerrillaFreeOpen(ctx context.Context, req *emptypb.Empty) (*pb.StartGuerrillaFreeOpenResponse, error) {
	log.Printf("[QuestService] StartGuerrillaFreeOpen")

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	s.users.UpdateUser(userId, func(user *store.UserState) {
		user.GuerrillaFreeOpen.StartDatetime = nowMillis
		user.GuerrillaFreeOpen.OpenMinutes = defaultGuerrillaFreeOpenMinutes
		user.GuerrillaFreeOpen.DailyOpenedCount++
		user.GuerrillaFreeOpen.LatestVersion = nowMillis
	})

	return &pb.StartGuerrillaFreeOpenResponse{}, nil
}
