package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type RewardServiceServer struct {
	pb.UnimplementedRewardServiceServer
	users     store.UserRepository
	sessions  store.SessionRepository
	bhCatalog *masterdata.BigHuntCatalog
	granter   *store.PossessionGranter
}

func NewRewardServiceServer(
	users store.UserRepository,
	sessions store.SessionRepository,
	bhCatalog *masterdata.BigHuntCatalog,
	granter *store.PossessionGranter,
) *RewardServiceServer {
	return &RewardServiceServer{users: users, sessions: sessions, bhCatalog: bhCatalog, granter: granter}
}

func (s *RewardServiceServer) ReceiveBigHuntReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceiveBigHuntRewardResponse, error) {
	log.Printf("[RewardService] ReceiveBigHuntReward")

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	weeklyVersion := gametime.WeeklyVersion(nowMillis)

	var weeklyScoreResults []*pb.WeeklyScoreResult
	var weeklyRewards []*pb.BigHuntReward
	isReceived := false

	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		ws := user.BigHuntWeeklyStatuses[weeklyVersion]
		isReceived = ws.IsReceivedWeeklyReward

		for _, boss := range s.bhCatalog.BossByBossId {
			key := store.BigHuntWeeklyScoreKey{
				BigHuntWeeklyVersion: weeklyVersion,
				AttributeType:        boss.AttributeType,
			}
			wms := user.BigHuntWeeklyMaxScores[key]
			gradeIcon := s.bhCatalog.ResolveGradeIconId(boss.BigHuntBossId, wms.MaxScore)
			weeklyScoreResults = append(weeklyScoreResults, &pb.WeeklyScoreResult{
				AttributeType:           boss.AttributeType,
				BeforeMaxScore:          wms.MaxScore,
				CurrentMaxScore:         wms.MaxScore,
				BeforeAssetGradeIconId:  gradeIcon,
				CurrentAssetGradeIconId: gradeIcon,
				AfterMaxScore:           wms.MaxScore,
				AfterAssetGradeIconId:   gradeIcon,
			})
		}

		if !isReceived {
			for _, boss := range s.bhCatalog.BossByBossId {
				rewardKey := masterdata.BigHuntWeeklyRewardKey{
					ScheduleId:    1,
					AttributeType: boss.AttributeType,
				}
				rewardGroupId := s.bhCatalog.ResolveActiveWeeklyRewardGroupId(rewardKey, nowMillis)
				if rewardGroupId == 0 {
					continue
				}

				weekKey := store.BigHuntWeeklyScoreKey{
					BigHuntWeeklyVersion: weeklyVersion,
					AttributeType:        boss.AttributeType,
				}
				maxScore := user.BigHuntWeeklyMaxScores[weekKey].MaxScore

				items := s.bhCatalog.CollectNewRewards(rewardGroupId, 0, maxScore)
				for _, item := range items {
					s.granter.GrantFull(user, model.PossessionType(item.PossessionType), item.PossessionId, item.Count, nowMillis)
					weeklyRewards = append(weeklyRewards, &pb.BigHuntReward{
						PossessionType: item.PossessionType,
						PossessionId:   item.PossessionId,
						Count:          item.Count,
					})
				}
			}

			ws.IsReceivedWeeklyReward = true
			ws.LatestVersion = nowMillis
			user.BigHuntWeeklyStatuses[weeklyVersion] = ws
			isReceived = true
		}
	})

	if weeklyRewards == nil {
		weeklyRewards = []*pb.BigHuntReward{}
	}
	if weeklyScoreResults == nil {
		weeklyScoreResults = []*pb.WeeklyScoreResult{}
	}

	tables := userdata.ProjectTables(user, []string{
		"IUserBigHuntWeeklyStatus",
		"IUserBigHuntWeeklyMaxScore",
		"IUserConsumableItem",
		"IUserMaterial",
	})

	return &pb.ReceiveBigHuntRewardResponse{
		WeeklyScoreResult:           weeklyScoreResults,
		WeeklyScoreReward:           weeklyRewards,
		IsReceivedWeeklyScoreReward: isReceived,
		LastWeekWeeklyScoreReward:   []*pb.BigHuntReward{},
		DiffUserData:                userdata.BuildDiffFromTables(tables),
	}, nil
}

func (s *RewardServiceServer) ReceivePvpReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceivePvpRewardResponse, error) {
	log.Printf("[RewardService] ReceivePvpReward (stub)")
	return &pb.ReceivePvpRewardResponse{
		DiffUserData: map[string]*pb.DiffData{},
	}, nil
}

func (s *RewardServiceServer) ReceiveLabyrinthSeasonReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceiveLabyrinthSeasonRewardResponse, error) {
	log.Printf("[RewardService] ReceiveLabyrinthSeasonReward (stub)")
	return &pb.ReceiveLabyrinthSeasonRewardResponse{
		DiffUserData: map[string]*pb.DiffData{},
	}, nil
}

func (s *RewardServiceServer) ReceiveMissionPassRemainingReward(ctx context.Context, _ *emptypb.Empty) (*pb.ReceiveMissionPassRemainingRewardResponse, error) {
	log.Printf("[RewardService] ReceiveMissionPassRemainingReward (stub)")
	return &pb.ReceiveMissionPassRemainingRewardResponse{
		DiffUserData: map[string]*pb.DiffData{},
	}, nil
}
