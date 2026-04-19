package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type BigHuntServiceServer struct {
	pb.UnimplementedBigHuntServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.BigHuntCatalog
	engine   *questflow.QuestHandler
}

func NewBigHuntServiceServer(
	users store.UserRepository,
	sessions store.SessionRepository,
	catalog *masterdata.BigHuntCatalog,
	engine *questflow.QuestHandler,
) *BigHuntServiceServer {
	return &BigHuntServiceServer{users: users, sessions: sessions, catalog: catalog, engine: engine}
}

var bigHuntDiffTables = []string{
	"IUserBigHuntProgressStatus",
	"IUserBigHuntMaxScore",
	"IUserBigHuntStatus",
	"IUserBigHuntScheduleMaxScore",
	"IUserBigHuntWeeklyMaxScore",
	"IUserBigHuntWeeklyStatus",
}

func buildBigHuntDiff(user store.UserState, tableNames []string) map[string]*pb.DiffData {
	tables := userdata.ProjectTables(user, tableNames)
	return userdata.BuildDiffFromTablesOrdered(tables, tableNames)
}

func (s *BigHuntServiceServer) StartBigHuntQuest(ctx context.Context, req *pb.StartBigHuntQuestRequest) (*pb.StartBigHuntQuestResponse, error) {
	log.Printf("[BigHuntService] StartBigHuntQuest: bossQuestId=%d questId=%d deckNumber=%d isDryRun=%v",
		req.BigHuntBossQuestId, req.BigHuntQuestId, req.UserDeckNumber, req.IsDryRun)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	bhQuest, ok := s.catalog.QuestById[req.BigHuntQuestId]
	if !ok {
		log.Printf("[BigHuntService] StartBigHuntQuest: unknown bigHuntQuestId=%d", req.BigHuntQuestId)
	}

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		if ok {
			s.engine.HandleBigHuntQuestStart(user, bhQuest.QuestId, req.UserDeckNumber, nowMillis)
		}

		user.BigHuntProgress = store.BigHuntProgress{
			CurrentBigHuntBossQuestId: req.BigHuntBossQuestId,
			CurrentBigHuntQuestId:     req.BigHuntQuestId,
			CurrentQuestSceneId:       0,
			IsDryRun:                  req.IsDryRun,
			LatestVersion:             nowMillis,
		}

		user.BigHuntDeckNumber = req.UserDeckNumber

		st := user.BigHuntStatuses[req.BigHuntBossQuestId]
		st.DailyChallengeCount++
		st.LatestChallengeDatetime = nowMillis
		st.LatestVersion = nowMillis
		user.BigHuntStatuses[req.BigHuntBossQuestId] = st
	})

	return &pb.StartBigHuntQuestResponse{
		DiffUserData: buildBigHuntDiff(user, append([]string{"IUserQuest"}, bigHuntDiffTables...)),
	}, nil
}

func (s *BigHuntServiceServer) UpdateBigHuntQuestSceneProgress(ctx context.Context, req *pb.UpdateBigHuntQuestSceneProgressRequest) (*pb.UpdateBigHuntQuestSceneProgressResponse, error) {
	log.Printf("[BigHuntService] UpdateBigHuntQuestSceneProgress: questSceneId=%d", req.QuestSceneId)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.BigHuntProgress.CurrentQuestSceneId = req.QuestSceneId
		user.BigHuntProgress.LatestVersion = nowMillis
	})

	return &pb.UpdateBigHuntQuestSceneProgressResponse{
		DiffUserData: buildBigHuntDiff(user, []string{"IUserBigHuntProgressStatus"}),
	}, nil
}

func (s *BigHuntServiceServer) FinishBigHuntQuest(ctx context.Context, req *pb.FinishBigHuntQuestRequest) (*pb.FinishBigHuntQuestResponse, error) {
	log.Printf("[BigHuntService] FinishBigHuntQuest: bossQuestId=%d questId=%d isRetired=%v",
		req.BigHuntBossQuestId, req.BigHuntQuestId, req.IsRetired)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	bhQuest := s.catalog.QuestById[req.BigHuntQuestId]
	bossQuest := s.catalog.BossQuestById[req.BigHuntBossQuestId]
	boss := s.catalog.BossByBossId[bossQuest.BigHuntBossId]

	var scoreInfo *pb.BigHuntScoreInfo
	var scoreRewards []*pb.BigHuntReward

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleBigHuntQuestFinish(user, bhQuest.QuestId, req.IsRetired, false, nowMillis)

		if req.IsRetired || user.BigHuntProgress.IsDryRun {
			user.BigHuntProgress = store.BigHuntProgress{LatestVersion: nowMillis}
			return
		}

		detail := user.BigHuntBattleDetail
		totalDamage := detail.TotalDamage
		baseScore := totalDamage

		difficultyBonusPermil := int32(0)
		if coeff, ok := s.catalog.ScoreCoefficients[bhQuest.BigHuntQuestScoreCoefficientId]; ok {
			difficultyBonusPermil = coeff
		}

		aliveBonusPermil := int32(500)

		maxComboBonusPermil := int32(0)
		if detail.MaxComboCount >= 100 {
			maxComboBonusPermil = 300
		} else if detail.MaxComboCount >= 50 {
			maxComboBonusPermil = 200
		} else if detail.MaxComboCount >= 20 {
			maxComboBonusPermil = 100
		}

		userScore := baseScore * int64(1000+difficultyBonusPermil+aliveBonusPermil+maxComboBonusPermil) / 1000

		isHighScore := false
		oldMaxBoss := user.BigHuntMaxScores[bossQuest.BigHuntBossId]
		oldMax := oldMaxBoss.MaxScore
		if userScore > oldMax {
			isHighScore = true
			user.BigHuntMaxScores[bossQuest.BigHuntBossId] = store.BigHuntMaxScore{
				MaxScore:               userScore,
				MaxScoreUpdateDatetime: nowMillis,
				LatestVersion:          nowMillis,
			}
		}

		schedKey := store.BigHuntScheduleScoreKey{
			BigHuntScheduleId: s.catalog.ActiveScheduleId,
			BigHuntBossId:     bossQuest.BigHuntBossId,
		}
		oldSchedMax := user.BigHuntScheduleMaxScores[schedKey].MaxScore
		if userScore > oldSchedMax {
			user.BigHuntScheduleMaxScores[schedKey] = store.BigHuntScheduleMaxScore{
				MaxScore:               userScore,
				MaxScoreUpdateDatetime: nowMillis,
				LatestVersion:          nowMillis,
			}
		}

		weeklyVersion := gametime.WeeklyVersion(nowMillis)
		weekKey := store.BigHuntWeeklyScoreKey{
			BigHuntWeeklyVersion: weeklyVersion,
			AttributeType:        boss.AttributeType,
		}
		oldWeeklyMax := user.BigHuntWeeklyMaxScores[weekKey].MaxScore
		if userScore > oldWeeklyMax {
			user.BigHuntWeeklyMaxScores[weekKey] = store.BigHuntWeeklyMaxScore{
				MaxScore:      userScore,
				LatestVersion: nowMillis,
			}
		}

		assetGradeIconId := s.catalog.ResolveGradeIconId(bossQuest.BigHuntBossId, userScore)

		scoreInfo = &pb.BigHuntScoreInfo{
			UserScore:             userScore,
			IsHighScore:           isHighScore,
			TotalDamage:           totalDamage,
			BaseScore:             baseScore,
			DifficultyBonusPermil: difficultyBonusPermil,
			AliveBonusPermil:      aliveBonusPermil,
			MaxComboBonusPermil:   maxComboBonusPermil,
			AssetGradeIconId:      assetGradeIconId,
		}

		if isHighScore {
			rewardGroupId := s.catalog.ResolveActiveScoreRewardGroupId(
				bossQuest.BigHuntScoreRewardGroupScheduleId, nowMillis)
			if rewardGroupId > 0 {
				newItems := s.catalog.CollectNewRewards(rewardGroupId, oldMax, userScore)
				for _, item := range newItems {
					s.engine.Granter.GrantFull(user, model.PossessionType(item.PossessionType), item.PossessionId, item.Count, nowMillis)
					scoreRewards = append(scoreRewards, &pb.BigHuntReward{
						PossessionType: item.PossessionType,
						PossessionId:   item.PossessionId,
						Count:          item.Count,
					})
				}
			}
		}

		user.BigHuntProgress = store.BigHuntProgress{LatestVersion: nowMillis}
		user.BigHuntBattleBinary = nil
		user.BigHuntBattleDetail = store.BigHuntBattleDetail{}
	})

	if scoreInfo == nil {
		scoreInfo = &pb.BigHuntScoreInfo{}
	}
	if scoreRewards == nil {
		scoreRewards = []*pb.BigHuntReward{}
	}

	return &pb.FinishBigHuntQuestResponse{
		ScoreInfo:   scoreInfo,
		ScoreReward: scoreRewards,
		BattleReport: &pb.BigHuntBattleReport{
			BattleReportWave: []*pb.BigHuntBattleReportWave{},
		},
		DiffUserData: buildBigHuntDiff(user, append([]string{
			"IUserQuest",
			"IUserConsumableItem",
			"IUserMaterial",
		}, bigHuntDiffTables...)),
	}, nil
}

func (s *BigHuntServiceServer) RestartBigHuntQuest(ctx context.Context, req *pb.RestartBigHuntQuestRequest) (*pb.RestartBigHuntQuestResponse, error) {
	log.Printf("[BigHuntService] RestartBigHuntQuest: bossQuestId=%d questId=%d", req.BigHuntBossQuestId, req.BigHuntQuestId)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	bhQuest := s.catalog.QuestById[req.BigHuntQuestId]

	var battleBinary []byte
	var deckNumber int32

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		s.engine.HandleBigHuntQuestStart(user, bhQuest.QuestId, user.BigHuntDeckNumber, nowMillis)

		user.BigHuntProgress.CurrentQuestSceneId = 0
		user.BigHuntProgress.LatestVersion = nowMillis

		st := user.BigHuntStatuses[req.BigHuntBossQuestId]
		st.DailyChallengeCount++
		st.LatestChallengeDatetime = nowMillis
		st.LatestVersion = nowMillis
		user.BigHuntStatuses[req.BigHuntBossQuestId] = st

		battleBinary = user.BigHuntBattleBinary
		deckNumber = user.BigHuntDeckNumber
	})

	return &pb.RestartBigHuntQuestResponse{
		BattleBinary: battleBinary,
		DeckNumber:   deckNumber,
		DiffUserData: buildBigHuntDiff(user, append([]string{"IUserQuest"}, bigHuntDiffTables...)),
	}, nil
}

func (s *BigHuntServiceServer) SkipBigHuntQuest(ctx context.Context, req *pb.SkipBigHuntQuestRequest) (*pb.SkipBigHuntQuestResponse, error) {
	log.Printf("[BigHuntService] SkipBigHuntQuest: bossQuestId=%d skipCount=%d", req.BigHuntBossQuestId, req.SkipCount)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		st := user.BigHuntStatuses[req.BigHuntBossQuestId]
		st.DailyChallengeCount += req.SkipCount
		st.LatestChallengeDatetime = nowMillis
		st.LatestVersion = nowMillis
		user.BigHuntStatuses[req.BigHuntBossQuestId] = st
	})

	return &pb.SkipBigHuntQuestResponse{
		ScoreReward:  []*pb.BigHuntReward{},
		DiffUserData: buildBigHuntDiff(user, bigHuntDiffTables),
	}, nil
}

func (s *BigHuntServiceServer) SaveBigHuntBattleInfo(ctx context.Context, req *pb.SaveBigHuntBattleInfoRequest) (*pb.SaveBigHuntBattleInfoResponse, error) {
	log.Printf("[BigHuntService] SaveBigHuntBattleInfo: elapsedFrames=%d", req.ElapsedFrameCount)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	var totalDamage int64
	if req.BigHuntBattleDetail != nil {
		for _, ci := range req.BigHuntBattleDetail.CostumeBattleInfo {
			if ci != nil {
				totalDamage += ci.TotalDamage
			}
		}
	}

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.BigHuntBattleBinary = req.BattleBinary

		if req.BigHuntBattleDetail != nil {
			user.BigHuntBattleDetail = store.BigHuntBattleDetail{
				DeckType:             req.BigHuntBattleDetail.DeckType,
				UserTripleDeckNumber: req.BigHuntBattleDetail.UserTripleDeckNumber,
				BossKnockDownCount:   req.BigHuntBattleDetail.BossKnockDownCount,
				MaxComboCount:        req.BigHuntBattleDetail.MaxComboCount,
				TotalDamage:          totalDamage,
			}
		}

		user.BigHuntProgress.LatestVersion = nowMillis
	})

	return &pb.SaveBigHuntBattleInfoResponse{
		DiffUserData: buildBigHuntDiff(user, []string{"IUserBigHuntProgressStatus"}),
	}, nil
}

func (s *BigHuntServiceServer) GetBigHuntTopData(ctx context.Context, _ *emptypb.Empty) (*pb.GetBigHuntTopDataResponse, error) {
	log.Printf("[BigHuntService] GetBigHuntTopData")

	userId := currentUserId(ctx, s.users, s.sessions)
	user, _ := s.users.LoadUser(userId)

	nowMillis := gametime.NowMillis()
	weeklyVersion := gametime.WeeklyVersion(nowMillis)

	var weeklyScoreResults []*pb.WeeklyScoreResult
	for _, boss := range s.catalog.BossByBossId {
		key := store.BigHuntWeeklyScoreKey{
			BigHuntWeeklyVersion: weeklyVersion,
			AttributeType:        boss.AttributeType,
		}
		ws := user.BigHuntWeeklyMaxScores[key]
		gradeIconId := s.catalog.ResolveGradeIconId(boss.BigHuntBossId, ws.MaxScore)

		weeklyScoreResults = append(weeklyScoreResults, &pb.WeeklyScoreResult{
			AttributeType:           boss.AttributeType,
			BeforeMaxScore:          ws.MaxScore,
			CurrentMaxScore:         ws.MaxScore,
			BeforeAssetGradeIconId:  gradeIconId,
			CurrentAssetGradeIconId: gradeIconId,
			AfterMaxScore:           ws.MaxScore,
			AfterAssetGradeIconId:   gradeIconId,
		})
	}

	ws := user.BigHuntWeeklyStatuses[weeklyVersion]

	weeklyRewards := s.resolveWeeklyRewards(user, weeklyVersion, nowMillis)

	lastWeekVersion := weeklyVersion - 7*24*60*60*1000
	lastWeekRewards := s.resolveWeeklyRewards(user, lastWeekVersion, nowMillis)

	return &pb.GetBigHuntTopDataResponse{
		WeeklyScoreResult:           weeklyScoreResults,
		WeeklyScoreReward:           weeklyRewards,
		IsReceivedWeeklyScoreReward: ws.IsReceivedWeeklyReward,
		LastWeekWeeklyScoreReward:   lastWeekRewards,
		DiffUserData:                buildBigHuntDiff(user, bigHuntDiffTables),
	}, nil
}

func (s *BigHuntServiceServer) resolveWeeklyRewards(user store.UserState, weeklyVersion, nowMillis int64) []*pb.BigHuntReward {
	var rewards []*pb.BigHuntReward
	for _, boss := range s.catalog.BossByBossId {
		rewardKey := masterdata.BigHuntWeeklyRewardKey{
			ScheduleId:    1,
			AttributeType: boss.AttributeType,
		}
		rewardGroupId := s.catalog.ResolveActiveWeeklyRewardGroupId(rewardKey, nowMillis)
		if rewardGroupId == 0 {
			continue
		}
		weekKey := store.BigHuntWeeklyScoreKey{
			BigHuntWeeklyVersion: weeklyVersion,
			AttributeType:        boss.AttributeType,
		}
		maxScore := user.BigHuntWeeklyMaxScores[weekKey].MaxScore
		for _, item := range s.catalog.CollectNewRewards(rewardGroupId, 0, maxScore) {
			rewards = append(rewards, &pb.BigHuntReward{
				PossessionType: item.PossessionType,
				PossessionId:   item.PossessionId,
				Count:          item.Count,
			})
		}
	}
	if rewards == nil {
		rewards = []*pb.BigHuntReward{}
	}
	return rewards
}
