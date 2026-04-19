package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

var materialDiffTables = []string{
	"IUserMaterial",
	"IUserConsumableItem",
}

type MaterialServiceServer struct {
	pb.UnimplementedMaterialServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.MaterialCatalog
	config   *masterdata.GameConfig
}

func NewMaterialServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.MaterialCatalog, config *masterdata.GameConfig) *MaterialServiceServer {
	return &MaterialServiceServer{users: users, sessions: sessions, catalog: catalog, config: config}
}

func (s *MaterialServiceServer) Sell(ctx context.Context, req *pb.MaterialSellRequest) (*pb.MaterialSellResponse, error) {
	log.Printf("[MaterialService] Sell: %d item(s)", len(req.MaterialPossession))

	userId := currentUserId(ctx, s.users, s.sessions)

	oldUser, _ := s.users.LoadUser(userId)
	tracker := userdata.NewDeleteTracker().
		Track("IUserMaterial", oldUser, userdata.SortedMaterialRecords, []string{"userId", "materialId"})

	snapshot, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		totalGold := int32(0)
		for _, item := range req.MaterialPossession {
			mat, ok := s.catalog.All[item.MaterialId]
			if !ok {
				log.Printf("[MaterialService] Sell: unknown materialId=%d, skipping", item.MaterialId)
				continue
			}

			cur := user.Materials[item.MaterialId]
			if cur < item.Count {
				log.Printf("[MaterialService] Sell: insufficient materialId=%d have=%d need=%d", item.MaterialId, cur, item.Count)
				continue
			}

			user.Materials[item.MaterialId] -= item.Count
			if user.Materials[item.MaterialId] <= 0 {
				delete(user.Materials, item.MaterialId)
			}

			gold := mat.SellPrice * item.Count
			totalGold += gold
			log.Printf("[MaterialService] Sell: materialId=%d x%d -> %d gold", item.MaterialId, item.Count, gold)
		}

		if totalGold > 0 {
			user.ConsumableItems[s.config.ConsumableItemIdForGold] += totalGold
			log.Printf("[MaterialService] Sell: total gold +%d", totalGold)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("material sell: %w", err)
	}

	tables := userdata.ProjectTables(snapshot, materialDiffTables)
	diff := tracker.Apply(snapshot, tables)

	return &pb.MaterialSellResponse{
		DiffUserData: diff,
	}, nil
}
