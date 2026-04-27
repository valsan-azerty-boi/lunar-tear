package service

import (
	"context"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
)

type CharacterServiceServer struct {
	pb.UnimplementedCharacterServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.CharacterRebirthCatalog
	config   *masterdata.GameConfig
}

func NewCharacterServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.CharacterRebirthCatalog, config *masterdata.GameConfig) *CharacterServiceServer {
	return &CharacterServiceServer{users: users, sessions: sessions, catalog: catalog, config: config}
}

func (s *CharacterServiceServer) Rebirth(ctx context.Context, req *pb.RebirthRequest) (*pb.RebirthResponse, error) {
	log.Printf("[CharacterService] Rebirth: characterId=%d rebirthCount=%d", req.CharacterId, req.RebirthCount)

	userId := CurrentUserId(ctx, s.users, s.sessions)
	nowMillis := gametime.NowMillis()

	stepGroupId, ok := s.catalog.StepGroupByCharacterId[req.CharacterId]
	if !ok {
		log.Printf("[CharacterService] Rebirth: no step group for characterId=%d", req.CharacterId)
		return &pb.RebirthResponse{}, nil
	}

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		current := user.CharacterRebirths[req.CharacterId]
		currentCount := current.RebirthCount
		targetCount := currentCount + req.RebirthCount

		for count := currentCount; count < targetCount; count++ {
			step, ok := s.catalog.StepByGroupAndCount[masterdata.StepKey{GroupId: stepGroupId, BeforeRebirthCount: count}]
			if !ok {
				log.Printf("[CharacterService] Rebirth: no step row for groupId=%d beforeCount=%d", stepGroupId, count)
				return
			}

			goldId := s.config.ConsumableItemIdForGold
			user.ConsumableItems[goldId] = max(user.ConsumableItems[goldId]-s.config.CharacterRebirthConsumeGold, 0)
			log.Printf("[CharacterService] Rebirth: consumed gold=%d", s.config.CharacterRebirthConsumeGold)

			materials := s.catalog.MaterialsByGroupId[step.CharacterRebirthMaterialGroupId]
			for _, mat := range materials {
				user.Materials[mat.MaterialId] -= mat.Count
				if user.Materials[mat.MaterialId] <= 0 {
					delete(user.Materials, mat.MaterialId)
				}
				log.Printf("[CharacterService] Rebirth: consumed material=%d count=%d", mat.MaterialId, mat.Count)
			}
		}

		log.Printf("[CharacterService] Rebirth: characterId=%d count %d -> %d", req.CharacterId, currentCount, targetCount)
		user.CharacterRebirths[req.CharacterId] = store.CharacterRebirthState{
			CharacterId:   req.CharacterId,
			RebirthCount:  targetCount,
			LatestVersion: nowMillis,
		}
	})
	if err != nil {
		log.Printf("[CharacterService] Rebirth error: %v", err)
		return nil, err
	}

	return &pb.RebirthResponse{}, nil
}
