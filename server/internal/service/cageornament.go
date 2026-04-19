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
)

type CageOrnamentServiceServer struct {
	pb.UnimplementedCageOrnamentServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.CageOrnamentCatalog
	granter  *store.PossessionGranter
}

func NewCageOrnamentServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.CageOrnamentCatalog, granter *store.PossessionGranter) *CageOrnamentServiceServer {
	return &CageOrnamentServiceServer{users: users, sessions: sessions, catalog: catalog, granter: granter}
}

func (s *CageOrnamentServiceServer) ReceiveReward(ctx context.Context, req *pb.ReceiveRewardRequest) (*pb.ReceiveRewardResponse, error) {
	log.Printf("[CageOrnamentService] ReceiveReward: cageOrnamentId=%d", req.CageOrnamentId)

	reward, ok := s.catalog.LookupReward(req.CageOrnamentId)
	if !ok {
		log.Fatalf("[CageOrnamentService] ReceiveReward: no reward for cageOrnamentId=%d", req.CageOrnamentId)
	}

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		user.CageOrnamentRewards[req.CageOrnamentId] = store.CageOrnamentRewardState{
			CageOrnamentId:      req.CageOrnamentId,
			AcquisitionDatetime: nowMillis,
			LatestVersion:       nowMillis,
		}
		s.granter.GrantFull(user, model.PossessionType(reward.PossessionType), reward.PossessionId, reward.Count, nowMillis)
	})

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(user,
		[]string{
			"IUserMaterial", "IUserConsumableItem", "IUserGem",
			"IUserCostume", "IUserCostumeActiveSkill", "IUserCharacter",
			"IUserWeapon", "IUserWeaponSkill", "IUserWeaponAbility",
			"IUserWeaponNote",
			"IUserCageOrnamentReward",
		},
	))
	userdata.AddWeaponStoryDiff(diff, user, s.granter.DrainChangedStoryWeaponIds())

	return &pb.ReceiveRewardResponse{
		CageOrnamentReward: []*pb.CageOrnamentReward{
			{
				PossessionType: reward.PossessionType,
				PossessionId:   reward.PossessionId,
				Count:          reward.Count,
			},
		},
		DiffUserData: diff,
	}, nil
}

func (s *CageOrnamentServiceServer) RecordAccess(ctx context.Context, req *pb.RecordAccessRequest) (*pb.RecordAccessResponse, error) {
	log.Printf("[CageOrnamentService] RecordAccess: cageOrnamentId=%d", req.CageOrnamentId)

	userId := currentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()
	user, _ := s.users.UpdateUser(userId, func(user *store.UserState) {
		if _, exists := user.CageOrnamentRewards[req.CageOrnamentId]; !exists {
			user.CageOrnamentRewards[req.CageOrnamentId] = store.CageOrnamentRewardState{
				CageOrnamentId:      req.CageOrnamentId,
				AcquisitionDatetime: nowMillis,
				LatestVersion:       nowMillis,
			}
		}
	})

	diff := userdata.BuildDiffFromTables(userdata.ProjectTables(user,
		[]string{"IUserCageOrnamentReward"},
	))

	return &pb.RecordAccessResponse{
		DiffUserData: diff,
	}, nil
}
