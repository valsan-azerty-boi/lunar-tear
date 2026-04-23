package service

import (
	"context"
	"fmt"
	"log"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
)

type ConsumableItemServiceServer struct {
	pb.UnimplementedConsumableItemServiceServer
	users    store.UserRepository
	sessions store.SessionRepository
	catalog  *masterdata.ConsumableItemCatalog
	config   *masterdata.GameConfig
}

func NewConsumableItemServiceServer(users store.UserRepository, sessions store.SessionRepository, catalog *masterdata.ConsumableItemCatalog, config *masterdata.GameConfig) *ConsumableItemServiceServer {
	return &ConsumableItemServiceServer{users: users, sessions: sessions, catalog: catalog, config: config}
}

func (s *ConsumableItemServiceServer) Sell(ctx context.Context, req *pb.ConsumableItemSellRequest) (*pb.ConsumableItemSellResponse, error) {
	log.Printf("[ConsumableItemService] Sell: %d item(s)", len(req.ConsumableItemPossession))

	userId := CurrentUserId(ctx, s.users, s.sessions)

	_, err := s.users.UpdateUser(userId, func(user *store.UserState) {
		totalGold := int32(0)
		for _, item := range req.ConsumableItemPossession {
			row, ok := s.catalog.All[item.ConsumableItemId]
			if !ok {
				log.Printf("[ConsumableItemService] Sell: unknown consumableItemId=%d, skipping", item.ConsumableItemId)
				continue
			}

			cur := user.ConsumableItems[item.ConsumableItemId]
			if cur < item.Count {
				log.Printf("[ConsumableItemService] Sell: insufficient consumableItemId=%d have=%d need=%d", item.ConsumableItemId, cur, item.Count)
				continue
			}

			user.ConsumableItems[item.ConsumableItemId] -= item.Count
			if user.ConsumableItems[item.ConsumableItemId] <= 0 {
				delete(user.ConsumableItems, item.ConsumableItemId)
			}

			gold := row.SellPrice * item.Count
			totalGold += gold
			log.Printf("[ConsumableItemService] Sell: consumableItemId=%d x%d -> %d gold", item.ConsumableItemId, item.Count, gold)
		}

		if totalGold > 0 {
			user.ConsumableItems[s.config.ConsumableItemIdForGold] += totalGold
			log.Printf("[ConsumableItemService] Sell: total gold +%d", totalGold)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("consumable item sell: %w", err)
	}

	return &pb.ConsumableItemSellResponse{}, nil
}
