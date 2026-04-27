package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

const (
	exploreStaminaRecovery  = 1000 // millivalue added on finish
	exploreRewardMaterialId = 100001
	exploreRewardBaseCount  = 1
)

type ExploreServiceServer struct {
	pb.UnimplementedExploreServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.ExploreCatalog
}

func NewExploreServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.ExploreCatalog) *ExploreServiceServer {
	return &ExploreServiceServer{users: users, sessions: sessions, catalog: catalog}
}

func (s *ExploreServiceServer) StartExplore(ctx context.Context, req *pb.StartExploreRequest) (*pb.StartExploreResponse, error) {
	log.Printf("[ExploreService] StartExplore: exploreId=%d useConsumableItemId=%d", req.ExploreId, req.UseConsumableItemId)

	if _, ok := s.catalog.Explores[req.ExploreId]; !ok {
		return nil, fmt.Errorf("explore id=%d not found", req.ExploreId)
	}

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		explore := s.catalog.Explores[req.ExploreId]
		if req.UseConsumableItemId > 0 && explore.ConsumeItemCount > 0 {
			cur := user.ConsumableItems[req.UseConsumableItemId]
			user.ConsumableItems[req.UseConsumableItemId] = cur - explore.ConsumeItemCount
			log.Printf("[ExploreService] StartExplore: consumed item=%d count=%d remaining=%d", req.UseConsumableItemId, explore.ConsumeItemCount, user.ConsumableItems[req.UseConsumableItemId])
		}

		user.Explore = store.ExploreState{
			PlayingExploreId:   req.ExploreId,
			IsUseExploreTicket: false,
			LatestPlayDatetime: nowMillis,
			LatestVersion:      nowMillis,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("start explore: %w", err)
	}

	return &pb.StartExploreResponse{}, nil
}

func (s *ExploreServiceServer) FinishExplore(ctx context.Context, req *pb.FinishExploreRequest) (*pb.FinishExploreResponse, error) {
	log.Printf("[ExploreService] FinishExplore: exploreId=%d score=%d", req.ExploreId, req.Score)

	explore, ok := s.catalog.Explores[req.ExploreId]
	if !ok {
		return nil, fmt.Errorf("explore id=%d not found", req.ExploreId)
	}

	assetGradeIconId := s.catalog.GradeForScore(req.ExploreId, req.Score)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	rewardCount := int32(exploreRewardBaseCount) * explore.RewardLotteryCount

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		existing, exists := user.ExploreScores[req.ExploreId]
		if !exists || req.Score > existing.MaxScore {
			user.ExploreScores[req.ExploreId] = store.ExploreScoreState{
				ExploreId:              req.ExploreId,
				MaxScore:               req.Score,
				MaxScoreUpdateDatetime: nowMillis,
				LatestVersion:          nowMillis,
			}
		}

		user.Explore = store.ExploreState{
			PlayingExploreId:   0,
			IsUseExploreTicket: false,
			LatestPlayDatetime: user.Explore.LatestPlayDatetime,
			LatestVersion:      nowMillis,
		}

		user.Status.StaminaMilliValue += exploreStaminaRecovery
		user.Status.StaminaUpdateDatetime = nowMillis
		user.Status.LatestVersion = nowMillis
		log.Printf("[ExploreService] FinishExplore: stamina +%d -> %d", exploreStaminaRecovery, user.Status.StaminaMilliValue)

		user.Materials[exploreRewardMaterialId] += rewardCount
		log.Printf("[ExploreService] FinishExplore: granted material=%d count=%d", exploreRewardMaterialId, rewardCount)
	})
	if err != nil {
		return nil, fmt.Errorf("finish explore: %w", err)
	}

	rewards := []*pb.ExploreReward{
		{
			PossessionType: int32(model.PossessionTypeMaterial),
			PossessionId:   exploreRewardMaterialId,
			Count:          rewardCount,
		},
	}

	return &pb.FinishExploreResponse{
		AcquireStaminaCount: exploreStaminaRecovery,
		ExploreReward:       rewards,
		AssetGradeIconId:    assetGradeIconId,
	}, nil
}

func (s *ExploreServiceServer) RetireExplore(ctx context.Context, req *pb.RetireExploreRequest) (*pb.RetireExploreResponse, error) {
	log.Printf("[ExploreService] RetireExplore: exploreId=%d", req.ExploreId)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.Explore = store.ExploreState{
			PlayingExploreId:   0,
			IsUseExploreTicket: false,
			LatestPlayDatetime: user.Explore.LatestPlayDatetime,
			LatestVersion:      nowMillis,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("retire explore: %w", err)
	}

	return &pb.RetireExploreResponse{}, nil
}
