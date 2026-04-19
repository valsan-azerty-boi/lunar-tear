package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

const companionMaxLevel = int32(50)

var companionDiffTables = []string{
	"IUserCompanion",
	"IUserMaterial",
	"IUserConsumableItem",
}

type CompanionServiceServer struct {
	pb.UnimplementedCompanionServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.CompanionCatalog
	config   *masterdata.GameConfig
}

func NewCompanionServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.CompanionCatalog, config *masterdata.GameConfig) *CompanionServiceServer {
	return &CompanionServiceServer{users: users, sessions: sessions, catalog: catalog, config: config}
}

func (s *CompanionServiceServer) Enhance(ctx context.Context, req *pb.CompanionEnhanceRequest) (*pb.CompanionEnhanceResponse, error) {
	log.Printf("[CompanionService] Enhance: uuid=%s addLevel=%d", req.UserCompanionUuid, req.AddLevelCount)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	snapshot, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		companion, ok := user.Companions[req.UserCompanionUuid]
		if !ok {
			log.Printf("[CompanionService] Enhance: companion uuid=%s not found", req.UserCompanionUuid)
			return
		}

		compDef, ok := s.catalog.CompanionById[companion.CompanionId]
		if !ok {
			log.Printf("[CompanionService] Enhance: companion master id=%d not found", companion.CompanionId)
			return
		}

		targetLevel := companion.Level + req.AddLevelCount
		if targetLevel > companionMaxLevel {
			targetLevel = companionMaxLevel
		}

		for lvl := companion.Level; lvl < targetLevel; lvl++ {
			if costFunc, ok := s.catalog.GoldCostByCategory[compDef.CompanionCategoryType]; ok {
				goldCost := costFunc.Evaluate(lvl)
				user.ConsumableItems[s.config.ConsumableItemIdForGold] -= goldCost
			}

			matKey := masterdata.CompanionLevelKey{CategoryType: compDef.CompanionCategoryType, Level: lvl}
			if mat, ok := s.catalog.MaterialsByKey[matKey]; ok {
				user.Materials[mat.MaterialId] -= mat.Count
			}
		}

		companion.Level = targetLevel
		companion.LatestVersion = nowMillis
		user.Companions[req.UserCompanionUuid] = companion
		log.Printf("[CompanionService] Enhance: companionId=%d level -> %d", companion.CompanionId, targetLevel)
	})
	if err != nil {
		return nil, fmt.Errorf("companion enhance: %w", err)
	}

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(snapshot, companionDiffTables))

	return &pb.CompanionEnhanceResponse{
		DiffUserData: diff,
	}, nil
}
