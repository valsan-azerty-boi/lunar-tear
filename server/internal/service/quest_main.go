package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type QuestServiceServer struct {
	pb.UnimplementedQuestServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	engine   *questflow.QuestHandler
}

func NewQuestServiceServer(users store.UserRepository, sessions store.SessionRepository, engine *questflow.QuestHandler) *QuestServiceServer {
	if engine == nil {
		panic("quest handler is required")
	}
	return &QuestServiceServer{users: users, sessions: sessions, engine: engine}
}

func buildSelectedQuestDiff(user store.UserState, tableNames []string) map[string]*pb.DiffData {
	tables := userdata.ProjectTables(user, tableNames)
	return userdata.BuildDiffFromTablesOrdered(tables, tableNames)
}

func (s *QuestServiceServer) UpdateMainFlowSceneProgress(ctx context.Context, req *pb.UpdateMainFlowSceneProgressRequest) (*pb.UpdateMainFlowSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateMainFlowSceneProgress: questSceneId=%d", req.QuestSceneId)

	userId := currentUserId(ctx, s.users, s.sessions)
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleMainFlowSceneProgress(user, req.QuestSceneId, gametime.NowMillis())
	})

	diff := buildSelectedQuestDiff(user, []string{
		"IUserMainQuestFlowStatus",
		"IUserMainQuestMainFlowStatus",
		"IUserMainQuestProgressStatus",
		"IUserMainQuestSeasonRoute",
		"IUserPortalCageStatus",
		"IUserSideStoryQuestSceneProgressStatus",
		"IUserQuest",
		"IUserCharacter",
		"IUserCostume",
		"IUserCostumeActiveSkill",
		"IUserWeapon",
		"IUserWeaponSkill",
		"IUserWeaponAbility",
		"IUserWeaponNote",
		"IUserCompanion",
		"IUserConsumableItem",
		"IUserMaterial",
		"IUserImportantItem",
		"IUserParts",
		"IUserPartsGroupNote",
	})
	userdata.AddWeaponStoryDiff(diff, user, s.engine.Granter.DrainChangedStoryWeaponIds())

	return &pb.UpdateMainFlowSceneProgressResponse{
		DiffUserData: diff,
	}, nil
}

func (s *QuestServiceServer) UpdateReplayFlowSceneProgress(ctx context.Context, req *pb.UpdateReplayFlowSceneProgressRequest) (*pb.UpdateReplayFlowSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateReplayFlowSceneProgress: questSceneId=%d", req.QuestSceneId)

	userId := currentUserId(ctx, s.users, s.sessions)
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleReplayFlowSceneProgress(user, req.QuestSceneId, gametime.NowMillis())
	})

	return &pb.UpdateReplayFlowSceneProgressResponse{
		DiffUserData: buildSelectedQuestDiff(user, []string{
			"IUserMainQuestFlowStatus",
			"IUserMainQuestReplayFlowStatus",
		}),
	}, nil
}

func (s *QuestServiceServer) UpdateMainQuestSceneProgress(ctx context.Context, req *pb.UpdateMainQuestSceneProgressRequest) (*pb.UpdateMainQuestSceneProgressResponse, error) {
	log.Printf("[QuestService] UpdateMainQuestSceneProgress: questSceneId=%d", req.QuestSceneId)

	userId := currentUserId(ctx, s.users, s.sessions)
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleMainQuestSceneProgress(user, req.QuestSceneId)
	})

	return &pb.UpdateMainQuestSceneProgressResponse{
		DiffUserData: buildSelectedQuestDiff(user, []string{
			"IUserStatus",
			"IUserCharacter",
			"IUserQuest",
			"IUserQuestMission",
			"IUserMainQuestFlowStatus",
			"IUserMainQuestMainFlowStatus",
			"IUserMainQuestProgressStatus",
		}),
	}, nil
}

func (s *QuestServiceServer) StartMainQuest(ctx context.Context, req *pb.StartMainQuestRequest) (*pb.StartMainQuestResponse, error) {
	log.Printf("[QuestService] StartMainQuest: %+v", req)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		if req.IsReplayFlow {
			s.engine.HandleQuestStartReplay(user, req.QuestId, req.IsBattleOnly, req.UserDeckNumber, nowMillis)
		} else {
			s.engine.HandleQuestStart(user, req.QuestId, req.IsBattleOnly, req.UserDeckNumber, nowMillis)
		}
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

	return &pb.StartMainQuestResponse{
		BattleDropReward: pbDrops,
		DiffUserData: buildSelectedQuestDiff(user, []string{
			"IUserStatus",
			"IUserQuest",
			"IUserQuestMission",
			"IUserMainQuestFlowStatus",
			"IUserMainQuestMainFlowStatus",
			"IUserMainQuestProgressStatus",
			"IUserMainQuestSeasonRoute",
			"IUserMainQuestReplayFlowStatus",
		}),
	}, nil
}

func toProtoRewards(grants []questflow.RewardGrant) []*pb.QuestReward {
	if len(grants) == 0 {
		return []*pb.QuestReward{}
	}
	out := make([]*pb.QuestReward, len(grants))
	for i, g := range grants {
		out[i] = &pb.QuestReward{
			PossessionType: int32(g.PossessionType),
			PossessionId:   g.PossessionId,
			Count:          g.Count,
		}
	}
	return out
}

func (s *QuestServiceServer) FinishMainQuest(ctx context.Context, req *pb.FinishMainQuestRequest) (*pb.FinishMainQuestResponse, error) {
	log.Printf("[QuestService] FinishMainQuest: questId=%d isMainFlow=%v isRetired=%v isAnnihilated=%v storySkipType=%d",
		req.QuestId, req.IsMainFlow, req.IsRetired, req.IsAnnihilated, req.StorySkipType)

	nowMillis := gametime.NowMillis()
	userId := currentUserId(ctx, s.users, s.sessions)
	var outcome questflow.FinishOutcome
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		outcome = s.engine.HandleQuestFinish(user, req.QuestId, req.IsRetired, req.IsAnnihilated, nowMillis)
	})

	diff := buildSelectedQuestDiff(user, []string{
		"IUserQuest",
		"IUserQuestMission",
		"IUserMainQuestFlowStatus",
		"IUserMainQuestMainFlowStatus",
		"IUserMainQuestProgressStatus",
		"IUserMainQuestSeasonRoute",
		"IUserMainQuestReplayFlowStatus",
		"IUserStatus",
		"IUserGem",
		"IUserCharacter",
		"IUserCostume",
		"IUserCostumeActiveSkill",
		"IUserWeapon",
		"IUserWeaponSkill",
		"IUserWeaponAbility",
		"IUserWeaponNote",
		"IUserCompanion",
		"IUserConsumableItem",
		"IUserMaterial",
		"IUserImportantItem",
		"IUserParts",
		"IUserPartsGroupNote",
	})
	userdata.AddWeaponStoryDiff(diff, user, outcome.ChangedWeaponStoryIds)

	return &pb.FinishMainQuestResponse{
		DropReward:                      toProtoRewards(outcome.DropRewards),
		FirstClearReward:                toProtoRewards(outcome.FirstClearRewards),
		MissionClearReward:              toProtoRewards(outcome.MissionClearRewards),
		MissionClearCompleteReward:      toProtoRewards(outcome.MissionClearCompleteRewards),
		AutoOrbitResult:                 []*pb.QuestReward{},
		IsBigWin:                        outcome.IsBigWin,
		BigWinClearedQuestMissionIdList: outcome.BigWinClearedQuestMissionIds,
		ReplayFlowFirstClearReward:      toProtoRewards(outcome.ReplayFlowFirstClearRewards),
		UserStatusCampaignReward:        []*pb.QuestReward{},
		DiffUserData:                    diff,
	}, nil
}

func (s *QuestServiceServer) RestartMainQuest(ctx context.Context, req *pb.RestartMainQuestRequest) (*pb.RestartMainQuestResponse, error) {
	log.Printf("[QuestService] RestartMainQuest: questId=%d isMainFlow=%v", req.QuestId, req.IsMainFlow)

	userId := currentUserId(ctx, s.users, s.sessions)
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleQuestRestart(user, req.QuestId, gametime.NowMillis())
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

	return &pb.RestartMainQuestResponse{
		BattleDropReward: pbDrops,
		DeckNumber:       user.Quests[req.QuestId].UserDeckNumber,
		DiffUserData: buildSelectedQuestDiff(user, []string{
			"IUserStatus",
			"IUserQuest",
			"IUserQuestMission",
			"IUserMainQuestFlowStatus",
			"IUserMainQuestMainFlowStatus",
			"IUserMainQuestProgressStatus",
			"IUserMainQuestSeasonRoute",
		}),
	}, nil
}

func (s *QuestServiceServer) FinishAutoOrbit(ctx context.Context, req *emptypb.Empty) (*pb.FinishAutoOrbitResponse, error) {
	log.Printf("[QuestService] FinishAutoOrbit")
	return &pb.FinishAutoOrbitResponse{
		DiffUserData: userdata.EmptyDiff(),
	}, nil
}

func (s *QuestServiceServer) SkipQuest(ctx context.Context, req *pb.SkipQuestRequest) (*pb.SkipQuestResponse, error) {
	log.Printf("[QuestService] SkipQuest: questId=%d skipCount=%d useEffectItems=%d", req.QuestId, req.SkipCount, len(req.UseEffectItem))

	nowMillis := gametime.NowMillis()
	userId := currentUserId(ctx, s.users, s.sessions)
	var outcome questflow.FinishOutcome
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		for _, item := range req.UseEffectItem {
			log.Printf("[QuestService] SkipQuest UseEffectItem: consumableItemId=%d count=%d", item.ConsumableItemId, item.Count)
			user.ConsumableItems[item.ConsumableItemId] -= item.Count
			if user.ConsumableItems[item.ConsumableItemId] < 0 {
				user.ConsumableItems[item.ConsumableItemId] = 0
			}
		}
		outcome = s.engine.HandleQuestSkip(user, req.QuestId, req.SkipCount, nowMillis)
	})

	return &pb.SkipQuestResponse{
		DropReward:               toProtoRewards(outcome.DropRewards),
		UserStatusCampaignReward: []*pb.QuestReward{},
		DiffUserData: buildSelectedQuestDiff(user, []string{
			"IUserQuest",
			"IUserStatus",
			"IUserConsumableItem",
			"IUserMaterial",
			"IUserParts",
			"IUserPartsGroupNote",
			"IUserCharacter",
			"IUserCostume",
		}),
	}, nil
}

func (s *QuestServiceServer) SetRoute(ctx context.Context, req *pb.SetRouteRequest) (*pb.SetRouteResponse, error) {
	log.Printf("[QuestService] SetRoute: mainQuestRouteId=%d", req.MainQuestRouteId)

	userId := currentUserId(ctx, s.users, s.sessions)
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.MainQuest.CurrentMainQuestRouteId = req.MainQuestRouteId
		if seasonId, ok := s.engine.SeasonIdByRouteId[req.MainQuestRouteId]; ok {
			user.MainQuest.MainQuestSeasonId = seasonId
		}
		now := gametime.NowMillis()
		user.PortalCageStatus.IsCurrentProgress = false
		user.PortalCageStatus.LatestVersion = now
	})

	return &pb.SetRouteResponse{
		DiffUserData: buildSelectedQuestDiff(user, []string{
			"IUserMainQuestSeasonRoute",
			"IUserMainQuestMainFlowStatus",
			"IUserPortalCageStatus",
		}),
	}, nil
}

func (s *QuestServiceServer) SetQuestSceneChoice(ctx context.Context, req *pb.SetQuestSceneChoiceRequest) (*pb.SetQuestSceneChoiceResponse, error) {
	log.Printf("[QuestService] SetQuestSceneChoice: questSceneId=%d choiceNumber=%d",
		req.QuestSceneId, req.ChoiceNumber)
	return &pb.SetQuestSceneChoiceResponse{
		DiffUserData: userdata.EmptyDiff(),
	}, nil
}

func (s *QuestServiceServer) ResetLimitContentQuestProgress(ctx context.Context, req *pb.ResetLimitContentQuestProgressRequest) (*pb.ResetLimitContentQuestProgressResponse, error) {
	log.Printf("[QuestService] ResetLimitContentQuestProgress: eventQuestChapterId=%d questId=%d",
		req.EventQuestChapterId, req.QuestId)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		if _, exists := user.SideStoryQuests[req.QuestId]; exists {
			user.SideStoryQuests[req.QuestId] = store.SideStoryQuestProgress{
				HeadSideStoryQuestSceneId: 0,
				SideStoryQuestStateType:   model.SideStoryQuestStateUnknown,
				LatestVersion:             nowMillis,
			}
		}

		delete(user.QuestLimitContentStatus, req.QuestId)

		if user.SideStoryActiveProgress.CurrentSideStoryQuestId == req.QuestId {
			user.SideStoryActiveProgress = store.SideStoryActiveProgress{
				LatestVersion: nowMillis,
			}
		}
	})

	return &pb.ResetLimitContentQuestProgressResponse{
		DiffUserData: buildSelectedQuestDiff(user, []string{
			"IUserSideStoryQuest",
			"IUserSideStoryQuestSceneProgressStatus",
			"IUserQuestLimitContentStatus",
		}),
	}, nil
}

func (s *QuestServiceServer) SetAutoSaleSetting(ctx context.Context, req *pb.SetAutoSaleSettingRequest) (*pb.SetAutoSaleSettingResponse, error) {
	log.Printf("[QuestService] SetAutoSaleSetting: items=%d", len(req.AutoSaleSettingItem))

	userId := currentUserId(ctx, s.users, s.sessions)
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.AutoSaleSettings = make(map[int32]store.AutoSaleSettingState, len(req.AutoSaleSettingItem))
		for itemType, itemValue := range req.AutoSaleSettingItem {
			user.AutoSaleSettings[itemType] = store.AutoSaleSettingState{
				PossessionAutoSaleItemType:  itemType,
				PossessionAutoSaleItemValue: itemValue,
			}
		}
	})

	return &pb.SetAutoSaleSettingResponse{
		DiffUserData: buildSelectedQuestDiff(user, []string{
			"IUserAutoSaleSettingDetail",
		}),
	}, nil
}
